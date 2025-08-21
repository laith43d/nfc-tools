package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

// min returns the smaller of two integers (for compatibility with older Go versions)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// APDU helpers
func transmit(card *scard.Card, apdu []byte) ([]byte, error) {
	resp, err := card.Transmit(apdu)
	if err != nil {
		return nil, err
	}
	if len(resp) < 2 {
		return nil, errors.New("short APDU response")
	}
	sw1 := resp[len(resp)-2]
	sw2 := resp[len(resp)-1]
	if sw1 != 0x90 || sw2 != 0x00 {
		return nil, fmt.Errorf("APDU failed: SW=%02X%02X", sw1, sw2)
	}
	return resp[:len(resp)-2], nil
}

// getUID uses the ACR/PCSC pseudo-APDU FF CA 00 00 00 to fetch UID
func getUID(card *scard.Card) ([]byte, error) {
	return transmit(card, []byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
}

// readPage reads one 4-byte page from a Type 2 tag using FF B0 00 <page> 04
func readPage(card *scard.Card, page byte) ([]byte, error) {
	return transmit(card, []byte{0xFF, 0xB0, 0x00, page, 0x04})
}

// readPageAlternative tries alternative methods to read a page if standard method fails
func readPageAlternative(card *scard.Card, page byte) ([]byte, error) {
	// Try standard method first
	if data, err := readPage(card, page); err == nil {
		return data, nil
	}

	// Try reading with different length
	if data, err := transmit(card, []byte{0xFF, 0xB0, 0x00, page, 0x10}); err == nil {
		// If we got 16 bytes, return just the first 4
		if len(data) >= 4 {
			return data[:4], nil
		}
		return data, nil
	}

	// Try reading without length specified
	if data, err := transmit(card, []byte{0xFF, 0xB0, 0x00, page}); err == nil {
		return data, nil
	}

	return nil, fmt.Errorf("all read methods failed for page %02X", page)
}

// identifyTagType attempts to identify the specific tag type
func identifyTagType(card *scard.Card) string {
	page0, err := readPage(card, 0x00)
	if err != nil {
		return "unknown"
	}

	if len(page0) >= 1 {
		uid0 := page0[0]
		// Check for NTAG213/215/216 by first UID byte and memory size
		switch uid0 {
		case 0x04:
			// Test memory boundaries to determine exact type
			if _, err := readPage(card, 0x2C); err != nil {
				return "NTAG213" // 180 bytes total, can't read beyond page 44 (0x2C)
			}
			if _, err := readPage(card, 0x86); err != nil {
				return "NTAG215" // 540 bytes total, can't read beyond page 134 (0x86)
			}
			return "NTAG216" // 930 bytes total
		default:
			return "Type2-compatible"
		}
	}
	return "unknown"
}

// analyzeNDEFStructure parses and explains NDEF TLV structure
func analyzeNDEFStructure(data []byte, startPage int) {
	fmt.Printf("\n=== NDEF TLV STRUCTURE ANALYSIS ===\n")

	if len(data) == 0 {
		fmt.Printf("No data to analyze\n")
		return
	}

	offset := 0
	pageOffset := startPage
	foundNDEF := false

	for offset < len(data) {
		tlvType := data[offset]
		currentPage := pageOffset + (offset / 4)
		byteInPage := offset % 4

		fmt.Printf("Page %02d, Byte %d: TLV Type = 0x%02X ", currentPage, byteInPage, tlvType)

		switch tlvType {
		case 0x00:
			fmt.Printf("(NULL/Padding)\n")
			offset++
		case 0x03:
			fmt.Printf("(NDEF Message)\n")
			foundNDEF = true
			if offset+1 >= len(data) {
				fmt.Printf("  ❌ Error: Missing length byte\n")
				offset++
				continue
			}

			length := data[offset+1]
			fmt.Printf("  Length: %d bytes\n", length)

			if length == 0 {
				fmt.Printf("  (Empty NDEF message)\n")
				offset += 2
				continue
			}

			if offset+2+int(length) > len(data) {
				fmt.Printf("  ❌ Error: NDEF length (%d) exceeds available data (%d bytes remaining)\n",
					length, len(data)-offset-2)
				// Try to parse what we have
				remaining := len(data) - offset - 2
				if remaining > 0 {
					ndefData := data[offset+2 : offset+2+remaining]
					fmt.Printf("  Partial NDEF Data: % X\n", ndefData)
					parseNDEFMessage(ndefData)
				}
				return
			}

			ndefData := data[offset+2 : offset+2+int(length)]
			fmt.Printf("  NDEF Data: % X\n", ndefData)
			parseNDEFMessage(ndefData)
			offset += 2 + int(length)
		case 0xFE:
			fmt.Printf("(Terminator)\n")
			if foundNDEF {
				fmt.Printf("✅ NDEF TLV structure complete\n")
			}
			return // Stop parsing at terminator
		default:
			if tlvType >= 0x01 && tlvType <= 0xFD {
				fmt.Printf("(Proprietary TLV)\n")
				// Try to read length if available
				if offset+1 < len(data) {
					length := data[offset+1]
					fmt.Printf("  Length: %d bytes\n", length)
					offset += 2 + int(length)
				} else {
					offset++
				}
			} else {
				fmt.Printf("(Invalid TLV type)\n")
				offset++
			}
		}
	}

	if !foundNDEF {
		fmt.Printf("⚠️  No NDEF TLV found in data area\n")
	}
}

// parseNDEFMessage parses NDEF message structure
func parseNDEFMessage(data []byte) {
	if len(data) == 0 {
		fmt.Printf("  (Empty NDEF message)\n")
		return
	}

	fmt.Printf("  === NDEF MESSAGE ANALYSIS ===\n")
	offset := 0
	recordNum := 1

	for offset < len(data) {
		fmt.Printf("    --- Record %d ---\n", recordNum)

		if offset >= len(data) {
			fmt.Printf("    ❌ Error: Unexpected end of data\n")
			break
		}

		// Parse NDEF record header
		header := data[offset]
		mb := (header & 0x80) != 0 // Message Begin
		me := (header & 0x40) != 0 // Message End
		cf := (header & 0x20) != 0 // Chunk Flag
		sr := (header & 0x10) != 0 // Short Record
		il := (header & 0x08) != 0 // ID Length present
		tnf := header & 0x07       // Type Name Format

		fmt.Printf("    Record Header: 0x%02X\n", header)
		fmt.Printf("      MB (Message Begin): %t\n", mb)
		fmt.Printf("      ME (Message End): %t\n", me)
		fmt.Printf("      CF (Chunk Flag): %t\n", cf)
		fmt.Printf("      SR (Short Record): %t\n", sr)
		fmt.Printf("      IL (ID Length): %t\n", il)
		fmt.Printf("      TNF (Type Name Format): %d (%s)\n", tnf, getTNFDescription(tnf))

		offset++
		if offset >= len(data) {
			fmt.Printf("    ❌ Error: Missing type length\n")
			break
		}

		// Type Length
		typeLength := data[offset]
		fmt.Printf("      Type Length: %d\n", typeLength)
		offset++

		// Payload Length
		var payloadLength uint32
		if sr {
			if offset >= len(data) {
				fmt.Printf("    ❌ Error: Missing payload length (short record)\n")
				break
			}
			payloadLength = uint32(data[offset])
			offset++
		} else {
			if offset+3 >= len(data) {
				fmt.Printf("    ❌ Error: Missing payload length (long record)\n")
				break
			}
			payloadLength = (uint32(data[offset]) << 24) |
				(uint32(data[offset+1]) << 16) |
				(uint32(data[offset+2]) << 8) |
				uint32(data[offset+3])
			offset += 4
		}
		fmt.Printf("      Payload Length: %d\n", payloadLength)

		// ID Length (if present)
		var idLength byte
		if il {
			if offset >= len(data) {
				fmt.Printf("    ❌ Error: Missing ID length\n")
				break
			}
			idLength = data[offset]
			fmt.Printf("      ID Length: %d\n", idLength)
			offset++
		}

		// Type
		var recordType []byte
		if typeLength > 0 {
			if offset+int(typeLength) > len(data) {
				fmt.Printf("    ❌ Error: Type length (%d) exceeds remaining data\n", typeLength)
				break
			}
			recordType = data[offset : offset+int(typeLength)]
			fmt.Printf("      Type: %s (% X)\n", string(recordType), recordType)
			offset += int(typeLength)
		} else {
			fmt.Printf("      Type: (none)\n")
		}

		// ID (if present)
		if il && idLength > 0 {
			if offset+int(idLength) > len(data) {
				fmt.Printf("    ❌ Error: ID length (%d) exceeds remaining data\n", idLength)
				break
			}
			id := data[offset : offset+int(idLength)]
			fmt.Printf("      ID: %s\n", string(id))
			offset += int(idLength)
		}

		// Payload
		if payloadLength > 0 {
			if offset+int(payloadLength) > len(data) {
				fmt.Printf("    ❌ Error: Payload length (%d) exceeds remaining data (%d bytes)\n",
					payloadLength, len(data)-offset)
				// Try to parse what we have
				remaining := len(data) - offset
				if remaining > 0 {
					payload := data[offset:]
					fmt.Printf("      Partial Payload: % X\n", payload)
					// Try to parse if it's a URI record
					if typeLength == 1 && len(recordType) > 0 && recordType[0] == 'U' {
						parseURIPayload(payload)
					}
				}
				break
			}

			payload := data[offset : offset+int(payloadLength)]
			fmt.Printf("      Payload: % X\n", payload)

			// Parse payload based on record type
			if typeLength == 1 && len(recordType) > 0 {
				switch recordType[0] {
				case 'U':
					parseURIPayload(payload)
				case 'T':
					fmt.Printf("        📝 Text Record\n")
					parseTextPayload(payload)
				default:
					fmt.Printf("        🔍 Unknown well-known type: %c\n", recordType[0])
				}
			}
			offset += int(payloadLength)
		}

		fmt.Printf("\n")
		recordNum++

		// If this was the last record (ME=true), stop parsing
		if me {
			fmt.Printf("    ✅ End of NDEF message\n")
			break
		}
	}
}

// parseURIPayload parses URI record payload
func parseURIPayload(payload []byte) {
	if len(payload) == 0 {
		fmt.Printf("        ❌ Empty URI payload\n")
		return
	}

	identifierCode := payload[0]
	prefix := getURIPrefix(identifierCode)

	if len(payload) > 1 {
		suffix := string(payload[1:])
		fullURI := prefix + suffix
		fmt.Printf("        🌐 URI: %s\n", fullURI)
		fmt.Printf("        Prefix Code: 0x%02X (%s)\n", identifierCode, prefix)
		fmt.Printf("        Suffix: %s\n", suffix)
	} else {
		if prefix != "" {
			fmt.Printf("        🌐 URI: %s\n", prefix)
			fmt.Printf("        Prefix Code: 0x%02X (%s)\n", identifierCode, prefix)
		} else {
			fmt.Printf("        ❌ URI has identifier but no suffix\n")
		}
	}
}

// parseTextPayload parses Text record payload
func parseTextPayload(payload []byte) {
	if len(payload) == 0 {
		fmt.Printf("        ❌ Empty text payload\n")
		return
	}

	// First byte contains language code length and UTF encoding flag
	statusByte := payload[0]
	utf16 := (statusByte & 0x80) != 0
	langCodeLen := int(statusByte & 0x3F)

	if len(payload) < 1+langCodeLen {
		fmt.Printf("        ❌ Invalid text record format\n")
		return
	}

	langCode := string(payload[1 : 1+langCodeLen])
	textData := payload[1+langCodeLen:]

	encoding := "UTF-8"
	if utf16 {
		encoding = "UTF-16"
	}

	fmt.Printf("        📝 Text: %s\n", string(textData))
	fmt.Printf("        Language: %s\n", langCode)
	fmt.Printf("        Encoding: %s\n", encoding)
}

// getTNFDescription returns human-readable TNF description
func getTNFDescription(tnf byte) string {
	switch tnf {
	case 0x00:
		return "Empty"
	case 0x01:
		return "Well-known"
	case 0x02:
		return "Media type"
	case 0x03:
		return "Absolute URI"
	case 0x04:
		return "External"
	case 0x05:
		return "Unknown"
	case 0x06:
		return "Unchanged"
	case 0x07:
		return "Reserved"
	default:
		return "Invalid"
	}
}

// getURIPrefix returns URI prefix for identifier code
func getURIPrefix(code byte) string {
	prefixes := map[byte]string{
		0x00: "",
		0x01: "http://www.",
		0x02: "https://www.",
		0x03: "http://",
		0x04: "https://",
		0x05: "tel:",
		0x06: "mailto:",
		0x07: "ftp://anonymous:anonymous@",
		0x08: "ftp://ftp.",
		0x09: "ftps://",
		0x0A: "sftp://",
		0x0B: "smb://",
		0x0C: "nfs://",
		0x0D: "ftp://",
		0x0E: "dav://",
		0x0F: "news:",
		0x10: "telnet://",
		0x11: "imap:",
		0x12: "rtsp://",
		0x13: "urn:",
		0x14: "pop:",
		0x15: "sip:",
		0x16: "sips:",
		0x17: "tftp:",
		0x18: "btspp://",
		0x19: "btl2cap://",
		0x1A: "btgoep://",
		0x1B: "tcpobex://",
		0x1C: "irdaobex://",
		0x1D: "file://",
		0x1E: "urn:epc:id:",
		0x1F: "urn:epc:tag:",
		0x20: "urn:epc:pat:",
		0x21: "urn:epc:raw:",
		0x22: "urn:epc:",
		0x23: "urn:nfc:",
	}

	if prefix, exists := prefixes[code]; exists {
		return prefix
	}
	return "Unknown prefix"
}

// analyzeLockBytes analyzes static and dynamic lock bytes
func analyzeLockBytes(card *scard.Card, tagType string) {
	fmt.Printf("\n=== LOCK BYTES ANALYSIS ===\n")

	// Static lock bytes (page 2, bytes 2-3)
	pg2, err := readPage(card, 0x02)
	if err == nil && len(pg2) == 4 {
		lock0 := pg2[2]
		lock1 := pg2[3]
		fmt.Printf("Static Lock Bytes (Page 2, bytes 2-3): %02X %02X\n", lock0, lock1)

		// Decode locked pages for Type 2 tags
		var lockedPages []string
		for i := 0; i < 16; i++ {
			var bitSet bool
			if i < 8 {
				bitSet = ((lock0 >> uint(i)) & 1) == 1
			} else {
				bitSet = ((lock1 >> uint(i-8)) & 1) == 1
			}
			if bitSet {
				lockedPages = append(lockedPages, fmt.Sprintf("%d", 3+i))
			}
		}

		if len(lockedPages) > 0 {
			fmt.Printf("  Locked pages: %s\n", strings.Join(lockedPages, ", "))
		} else {
			fmt.Printf("  No pages locked by static lock bytes\n")
		}
	}

	// Dynamic lock bytes for NTAG
	if strings.Contains(tagType, "NTAG") {
		dynamicLockPage := byte(0x2A) // NTAG213 dynamic lock page
		if tagType == "NTAG215" {
			dynamicLockPage = 0x82
		} else if tagType == "NTAG216" {
			dynamicLockPage = 0xE2
		}

		if dynLock, err := readPage(card, dynamicLockPage); err == nil {
			fmt.Printf("Dynamic Lock Bytes (Page %02X): % X\n", dynamicLockPage, dynLock)
		}

		// Configuration pages
		configPage := dynamicLockPage + 1
		if cfg, err := readPage(card, configPage); err == nil {
			fmt.Printf("Configuration (Page %02X): % X\n", configPage, cfg)
			if len(cfg) >= 4 {
				fmt.Printf("  MIRROR: %02X\n", cfg[0])
				fmt.Printf("  RFUI: %02X\n", cfg[1])
				fmt.Printf("  MIRROR_PAGE: %02X\n", cfg[2])
				fmt.Printf("  AUTH0: %02X", cfg[3])
				if cfg[3] == 0xFF {
					fmt.Printf(" (password protection disabled)")
				} else {
					fmt.Printf(" (password protection starts at page %d)", cfg[3])
				}
				fmt.Printf("\n")
			}
		}
	}
}

// readFullTag reads and analyzes the complete NFC tag structure
func readFullTag(card *scard.Card) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("COMPREHENSIVE NFC TAG ANALYSIS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	// Get UID
	uid, err := getUID(card)
	if err != nil {
		fmt.Printf("❌ Failed to get UID: %v\n", err)
		return
	}
	uidHex := strings.ToUpper(hex.EncodeToString(uid))
	fmt.Printf("🏷️  Tag UID: %s\n", uidHex)

	// Identify tag type
	tagType := identifyTagType(card)
	fmt.Printf("📋 Tag Type: %s\n", tagType)

	// Determine memory layout
	var maxPage byte = 0x10 // Default for basic Type 2
	switch tagType {
	case "NTAG213":
		maxPage = 0x2C
	case "NTAG215":
		maxPage = 0x86
	case "NTAG216":
		maxPage = 0xE7
	}

	fmt.Printf("💾 Memory Layout: %d pages (0x00 to 0x%02X)\n", maxPage+1, maxPage)

	// Read header pages (0-3)
	fmt.Printf("\n=== HEADER PAGES (0-3) ===\n")
	for page := byte(0x00); page <= 0x03; page++ {
		data, err := readPage(card, page)
		if err != nil {
			fmt.Printf("Page %02d: ❌ Error: %v", page, err)
			// Try alternative method for header pages
			if altData, altErr := readPageAlternative(card, page); altErr == nil {
				fmt.Printf("Page %02d: ✅ Alternative read: % X", page, altData)
				data = altData
				err = nil
			} else {
				fmt.Printf("Page %02d: ❌ All read methods failed\n", page)
				continue
			}
		}

		if err == nil {
			fmt.Printf("Page %02d: % X", page, data)
			switch page {
			case 0x00:
				fmt.Printf(" (UID part 1)")
				if len(data) >= 3 {
					fmt.Printf("\n    Manufacturer: %02X", data[0])
					fmt.Printf("\n    UID bytes: %02X %02X", data[1], data[2])
					if len(data) >= 4 {
						fmt.Printf(" %02X", data[3])
					}
				}
			case 0x01:
				fmt.Printf(" (UID part 2)")
			case 0x02:
				if len(data) >= 4 {
					fmt.Printf(" (UID part 3 + Lock bytes: %02X %02X)", data[2], data[3])
					fmt.Printf("\n    Internal: %02X", data[1])
					fmt.Printf("\n    Static Lock 0: %02X", data[2])
					fmt.Printf("\n    Static Lock 1: %02X", data[3])
				}
			case 0x03:
				fmt.Printf(" (Capability Container - CC)")
				if len(data) >= 4 {
					magic1, magic2, size, access := data[0], data[1], data[2], data[3]
					fmt.Printf("\n    Magic: %02X %02X", magic1, magic2)
					fmt.Printf("\n    Size: %02X (data area = %d bytes)", size, int(size)*8)
					fmt.Printf("\n    Access: %02X", access)

					if magic1 == 0xE1 && magic2 == 0x10 {
						fmt.Printf("\n    ✅ Valid NDEF CC (Type 2 Tag)")
					} else if magic1 == 0xE1 && magic2 == 0x11 {
						fmt.Printf("\n    ✅ Valid NDEF CC (Type 4 Tag)")
					} else if magic1 == 0xE1 {
						fmt.Printf("\n    ⚠️  NDEF CC with non-standard version (%02X)", magic2)
					} else {
						fmt.Printf("\n    ❌ Invalid or non-NDEF CC")
						if magic1 == 0x00 && magic2 == 0x00 && size == 0x00 && access == 0x00 {
							fmt.Printf(" (appears to be empty/unformatted)")
						}
					}
				}
			}
			fmt.Printf("\n")
		}
	}

	// Read and analyze NDEF data area
	fmt.Printf("\n=== NDEF DATA AREA (Pages 4+) ===\n")
	var allNDEFData []byte
	startDataPage := 4

	// Read pages until we hit terminator or max pages
	consecutiveErrors := 0
	for page := byte(startDataPage); page <= maxPage; page++ {
		data, err := readPage(card, page)
		if err != nil {
			fmt.Printf("Page %02d: ❌ Error: %v", page, err)
			// Try alternative reading method
			if altData, altErr := readPageAlternative(card, page); altErr == nil {
				fmt.Printf("Page %02d: ✅ Alternative read: % X\n", page, altData)
				data = altData
				err = nil
			} else {
				fmt.Printf("Page %02d: ❌ Alternative read also failed: %v\n", page, altErr)
				consecutiveErrors++
				// If we can't read beyond a certain point, we might have hit memory boundary
				// But continue trying a few more pages in case it's a temporary issue
				if consecutiveErrors >= 3 || page > maxPage-5 {
					fmt.Printf("  (Stopping due to consecutive read errors - likely memory boundary)\n")
					break
				}
				continue
			}
		}

		if err == nil {
			consecutiveErrors = 0 // Reset error counter on successful read
			fmt.Printf("Page %02d: % X\n", page, data)
			allNDEFData = append(allNDEFData, data...)

			// Stop if we hit terminator TLV
			for _, b := range data {
				if b == 0xFE {
					goto analyzeNDEF
				}
			}
		}
	}

analyzeNDEF:
	// Analyze NDEF structure
	if len(allNDEFData) > 0 {
		analyzeNDEFStructure(allNDEFData, startDataPage)
	} else {
		fmt.Printf("⚠️  No NDEF data found in standard location (pages 4+)\n")
		fmt.Printf("🔍 Attempting to scan entire memory for NDEF patterns...\n")

		// Try to find NDEF data in other locations
		foundAlternativeData := false
		for page := byte(0x00); page <= maxPage; page++ {
			if data, err := readPageAlternative(card, page); err == nil {
				// Look for NDEF TLV pattern (0x03)
				for i, b := range data {
					if b == 0x03 && i+1 < len(data) {
						length := data[i+1]
						fmt.Printf("🎯 Found NDEF TLV at page %02X, byte %d (length: %d)\n", page, i, length)
						foundAlternativeData = true

						// Try to read the NDEF data from this location
						var ndefData []byte
						remainingInPage := len(data) - i - 2
						if int(length) <= remainingInPage {
							ndefData = data[i+2 : i+2+int(length)]
						} else {
							// Data spans multiple pages
							ndefData = append(ndefData, data[i+2:]...)
							bytesNeeded := int(length) - remainingInPage

							for nextPage := page + 1; bytesNeeded > 0 && nextPage <= maxPage; nextPage++ {
								if nextData, err := readPageAlternative(card, nextPage); err == nil {
									take := min(bytesNeeded, len(nextData))
									ndefData = append(ndefData, nextData[:take]...)
									bytesNeeded -= take
								} else {
									break
								}
							}
						}

						if len(ndefData) > 0 {
							fmt.Printf("🔍 Alternative NDEF Data: % X\n", ndefData)
							parseNDEFMessage(ndefData)
						}
						break
					}
				}
			}
		}

		if !foundAlternativeData {
			fmt.Printf("❌ No NDEF data found anywhere on the tag\n")
		}
	}

	// Analyze lock bytes
	analyzeLockBytes(card, tagType)

	// Show configuration pages for NTAG
	if strings.Contains(tagType, "NTAG") {
		fmt.Printf("\n=== NTAG CONFIGURATION PAGES ===\n")
		configStart := byte(0x29) // NTAG213
		if tagType == "NTAG215" {
			configStart = 0x83
		} else if tagType == "NTAG216" {
			configStart = 0xE3
		}

		for i := byte(0); i < 4; i++ {
			page := configStart + i
			if page <= maxPage {
				data, err := readPage(card, page)
				if err != nil {
					fmt.Printf("Page %02X: ❌ Error: %v\n", page, err)
				} else {
					fmt.Printf("Page %02X: % X", page, data)
					switch i {
					case 1:
						fmt.Printf(" (Dynamic Lock)")
					case 2:
						fmt.Printf(" (Configuration)")
					case 3:
						fmt.Printf(" (Password)")
					}
					fmt.Printf("\n")
				}
			}
		}
	}

	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("✅ ANALYSIS COMPLETE\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
}

// showIdealNFCFormat demonstrates what a properly formatted NFC tag should look like
func showIdealNFCFormat() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("IDEAL NFC TAG FORMAT STRUCTURE\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	fmt.Printf(`
📋 NTAG213 MEMORY LAYOUT (180 bytes total, 45 pages of 4 bytes each):

=== HEADER PAGES (0-3) - FACTORY SET ===
Page 00: [UID0][UID1][UID2][BCC0]     // UID part 1 + checksum
Page 01: [UID3][UID4][UID5][UID6]     // UID part 2  
Page 02: [BCC1][INT][LOCK0][LOCK1]    // Checksum + Internal + Static locks
Page 03: [E1][10][SIZE][ACCESS]       // Capability Container (CC)

🔧 CAPABILITY CONTAINER (Page 3) BREAKDOWN:
  E1 = Magic number (NDEF compatible Type 2 tag)
  10 = Version (1.0)
  SIZE = Data area size in 8-byte units (0x12 = 18*8 = 144 bytes for NTAG213)
  ACCESS = Access conditions (0x00 = read/write allowed)

=== NDEF DATA AREA (Pages 4-39) ===
Page 04: [03][LEN][NDEF...][FE]       // TLV: Type=03(NDEF), Length, Data, Terminator
Page 05+: [NDEF payload continues...]  // Additional NDEF data

🔧 TLV STRUCTURE:
  03 = TLV Type (NDEF Message)
  LEN = Length of NDEF message (1 byte for messages <255 bytes)
  NDEF... = NDEF message payload
  FE = Terminator TLV

📝 NDEF RECORD FORMAT (for URI):
  [HEADER][TYPE_LEN][PAYLOAD_LEN][TYPE][PAYLOAD]
  
  HEADER byte breakdown:
    Bit 7 (MB): Message Begin = 1
    Bit 6 (ME): Message End = 1  
    Bit 5 (CF): Chunk Flag = 0
    Bit 4 (SR): Short Record = 1
    Bit 3 (IL): ID Length = 0
    Bits 2-0 (TNF): Type Name Format = 001 (Well-known)
    Result: 11010001 = 0xD1

  TYPE_LEN = 0x01 (1 byte)
  PAYLOAD_LEN = Length of URI payload
  TYPE = 'U' (0x55) for URI record
  PAYLOAD = [URI_CODE][URI_STRING]

🌐 URI PAYLOAD:
  URI_CODE examples:
    0x00 = No prefix
    0x01 = "http://www."
    0x02 = "https://www."  
    0x03 = "http://"
    0x04 = "https://"
  
  Example for "https://example.com":
    URI_CODE = 0x04 ("https://")
    URI_STRING = "example.com"

=== CONFIGURATION AREA (Pages 40-44) ===
Page 40: [MIRROR][RFUI][MIRROR_PAGE][AUTH0]  // Mirror config
Page 41: [LOCK2][LOCK3][LOCK4][RFUI]         // Dynamic lock bytes
Page 42: [CFG0][CFG1][CFG2][CFG3]            // Configuration  
Page 43: [PWD0][PWD1][PWD2][PWD3]            // Password
Page 44: [PACK0][PACK1][RFUI][RFUI]          // Password acknowledge

🔧 KEY CONFIGURATION BYTES:
  AUTH0 (Page 40, byte 3): 
    0xFF = No password protection
    0x04-0x27 = Password required starting from this page
  
  CFG0 (Page 42, byte 0): Mirror configuration
  ACCESS (Page 3, byte 3): Access permissions for data area

✅ EXAMPLE: FORMATTED TAG WITH "https://example.com"
Page 03: E1 10 12 00                 // Valid CC
Page 04: 03 0F D1 01 0B 55 04 65 78 61 6D 70 6C 65 2E 63  // TLV + NDEF
Page 05: 6F 6D FE 00                 // "om" + Terminator + padding
`)

	fmt.Printf(strings.Repeat("=", 60) + "\n")
}

func main() {
	log.SetFlags(0)

	// Check for demo mode
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		showIdealNFCFormat()
		return
	}

	// Establish PC/SC context
	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatalf("pcsc EstablishContext: %v", err)
	}
	defer ctx.Release()

	// Ensure a reader is available
	readers, err := ctx.ListReaders()
	if err != nil {
		log.Fatalf("pcsc ListReaders: %v", err)
	}
	if len(readers) == 0 {
		log.Fatalf("no PC/SC readers found")
	}
	reader := readers[0]
	fmt.Printf("📱 Using reader: %s\n", reader)
	fmt.Printf("🔄 Waiting for NFC tags... (place tag on reader)\n\n")

	// Loop forever: wait for insertion, process, then wait for removal
	for {
		// Wait until a card is present
		waitForCardPresent(ctx, reader)

		// Try connecting
		var card *scard.Card
		for i := 0; i < 10; i++ {
			card, err = ctx.Connect(reader, scard.ShareExclusive, scard.ProtocolAny)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			fmt.Printf("❌ Connect failed: %v\n", err)
			waitForCardRemoval(ctx, reader)
			continue
		}

		// Process the tag
		func() {
			defer card.Disconnect(scard.LeaveCard)
			readFullTag(card)
		}()

		// Wait until the card is removed before processing the next one
		fmt.Printf("\n🔄 Remove tag and place another to analyze...\n\n")
		waitForCardRemoval(ctx, reader)
	}
}

// waitForCardPresent blocks until the reader reports a present card
func waitForCardPresent(ctx *scard.Context, reader string) {
	rs := []scard.ReaderState{{Reader: reader, CurrentState: scard.StateUnaware}}
	for {
		_ = ctx.GetStatusChange(rs, time.Second)
		st := rs[0].EventState
		rs[0].CurrentState = st
		if st&scard.StatePresent != 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// waitForCardRemoval blocks until the reader reports no card present
func waitForCardRemoval(ctx *scard.Context, reader string) {
	rs := []scard.ReaderState{{Reader: reader, CurrentState: scard.StateUnaware}}
	for {
		_ = ctx.GetStatusChange(rs, time.Second)
		st := rs[0].EventState
		rs[0].CurrentState = st
		if st&scard.StatePresent == 0 {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
}
