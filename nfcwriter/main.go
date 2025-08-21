package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

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

// writePage writes one 4-byte page to a Type 2 tag using FF D6 00 <page> 04 + data
func writePage(card *scard.Card, page byte, data []byte) error {
	if len(data) != 4 {
		return fmt.Errorf("page write must be 4 bytes, got %d", len(data))
	}
	apdu := append([]byte{0xFF, 0xD6, 0x00, page, 0x04}, data...)
	_, err := transmit(card, apdu)
	return err
}

// buildURIRecord builds a single-record NDEF message for a URI using SR
// URI payload = [identifierCode][uriWithoutPrefix]
// identifierCode 0x04 = "https://"
func buildURIRecord(uri string) []byte {
	payload := []byte{}
	// Normalize and choose identifier code 0x04 (https://)
	trimmed := strings.TrimPrefix(strings.TrimPrefix(uri, "http://"), "https://")
	payload = append(payload, 0x04)
	payload = append(payload, []byte(trimmed)...)

	// NDEF short record header
	header := byte(0xD1)  // MB=1, ME=1, SR=1, TNF=0x01 (Well-known)
	typeLen := byte(0x01) // 'U'
	payloadLen := byte(len(payload))
	typ := byte('U')

	msg := []byte{header, typeLen, payloadLen, typ}
	msg = append(msg, payload...)
	return msg
}

// formatType2Tag formats an NFC card according to NFC Forum Type 2 data format
// This initializes the capability container and prepares the tag for NDEF writing
func formatType2Tag(card *scard.Card) error {
	// Page 0: Manufacturer data (UID) - read-only, don't modify
	// Page 1: Reserved for manufacturer - don't modify

	// Page 2: Lock bytes (bytes 2-3 of page 2)
	// Setting to 0x00, 0x00 means no blocks are locked
	lockBytes := []byte{0x00, 0x00, 0x00, 0x00}
	if err := writePage(card, 0x02, lockBytes); err != nil {
		return fmt.Errorf("write lock bytes: %w", err)
	}

	// Page 3: Capability Container (CC)
	// Byte 0: Magic number (0xE1) - indicates NDEF capability
	// Byte 1: Version (0x10) - version 1.0
	// Byte 2: Data size (0x3F) - 504 bytes available (0x3F * 8 = 504)
	// Byte 3: Access conditions (0x00) - read/write allowed
	cc := []byte{0xE1, 0x10, 0x3F, 0x00}
	if err := writePage(card, 0x03, cc); err != nil {
		return fmt.Errorf("write capability container: %w", err)
	}

	// Page 4 and beyond: Clear NDEF data area
	// Initialize with NULL TLV (0x00) and then terminator TLV (0xFE)
	// This ensures the tag is properly formatted but empty
	clearData := []byte{0x00, 0x00, 0x00, 0xFE}
	if err := writePage(card, 0x04, clearData); err != nil {
		return fmt.Errorf("write initial NDEF area: %w", err)
	}

	return nil
}

// writeNDEFToType2 writes TLV (0x03, len, ndef...) and terminator 0xFE starting at page 4
func writeNDEFToType2(card *scard.Card, ndef []byte) error {
	if len(ndef) > 254 {
		return fmt.Errorf("NDEF too large for single-byte TLV length: %d", len(ndef))
	}
	tlv := []byte{0x03, byte(len(ndef))}
	tlv = append(tlv, ndef...)
	tlv = append(tlv, 0xFE)

	// Write starting at page 4, 4 bytes per page
	page := byte(0x04)
	// Ensure data length is multiple of 4 by padding 0x00
	pad := (4 - (len(tlv) % 4)) % 4
	if pad > 0 {
		tlv = append(tlv, make([]byte, pad)...)
	}
	for i := 0; i < len(tlv); i += 4 {
		if err := writePage(card, page, tlv[i:i+4]); err != nil {
			return fmt.Errorf("write page %d: %w", page, err)
		}
		page++
	}
	return nil
}

func main() {
	log.SetFlags(0)

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
	log.Printf("Using reader: %s", reader)

	// Loop forever: wait for insertion, process, then wait for removal
	for {
		// Wait until a card is present
		waitForCardPresent(ctx, reader)

		// Try connecting (retry briefly on transient errors)
		var card *scard.Card
		for i := 0; i < 10; i++ {
			card, err = ctx.Connect(reader, scard.ShareExclusive, scard.ProtocolAny)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			log.Printf("connect failed: %v", err)
			waitForCardRemoval(ctx, reader)
			continue
		}

		// Process the tag
		func() {
			defer card.Disconnect(scard.LeaveCard)

			// Get UID
			uid, err := getUID(card)
			if err != nil {
				log.Printf("get UID: %v", err)
				return
			}
			uidHex := strings.ToUpper(hex.EncodeToString(uid))
			log.Printf("Tag UID: %s", uidHex)

			// Format the card as NFC Forum Type 2 format
			log.Printf("Formatting tag as NFC Forum Type 2...")
			if err := formatType2Tag(card); err != nil {
				log.Printf("format Type 2 tag failed: %v", err)
				return
			}
			log.Printf("Tag formatted successfully")

			// Small delay after formatting as requested
			time.Sleep(200 * time.Millisecond)

			// Build URL and NDEF
			fullURL := fmt.Sprintf("https://dnd.qrand.me/r/%s", uidHex)
			ndef := buildURIRecord(fullURL)

			// Write NDEF directly to memory
			if err := writeNDEFToType2(card, ndef); err != nil {
				log.Printf("write NDEF failed: %v", err)
				return
			}
			log.Printf("Wrote URL to tag: %s", fullURL)
		}()

		// Wait until the card is removed before processing the next one
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
