# NFC Tools Suite

A collection of Go-based NFC (Near Field Communication) applications for reading, writing, and processing NFC tags. This suite provides comprehensive NFC functionality including tag analysis, URL writing, and UID clipboard services.

## Applications

This repository contains three distinct NFC applications, each serving a specific purpose in NFC tag management and processing.

### 1. NFC Reader (`nfcreader/`)

A comprehensive NFC tag analysis tool that reads and analyzes the complete structure of NFC tags.

#### Features
- 🔍 **Detailed Tag Analysis**: Reads and analyzes NFC tag memory structure
- 📋 **NDEF Message Parsing**: Decodes NDEF (NFC Data Exchange Format) messages
- 🔒 **Lock Byte Analysis**: Analyzes static and dynamic lock bytes
- 📊 **Memory Layout**: Displays complete tag memory structure
- 🏷️ **Tag Type Identification**: Automatically identifies tag types (NTAG213/215/216)
- 📝 **TLV Structure Analysis**: Parses Type-Length-Value data structures

#### Usage
```bash
cd nfcreader
go run main.go
```

#### Demo Mode
```bash
cd nfcreader
go run main.go demo
```

#### What it does
- Reads NFC tag UIDs and complete memory contents
- Analyzes NDEF data structure and content
- Identifies tag manufacturer and type
- Shows lock byte configuration
- Parses URI records and text records
- Displays comprehensive memory layout analysis

#### Supported Tag Types
- NTAG213 (180 bytes memory)
- NTAG215 (540 bytes memory)
- NTAG216 (930 bytes memory)
- Generic NFC Forum Type 2 compatible tags

### 2. NFC Writer (`nfcwriter/`)

An NFC tag writer that formats tags and writes URL data based on the tag's unique identifier.

#### Features
- ✏️ **Tag Writing**: Formats and writes NDEF data to NFC tags
- 🌐 **URL Generation**: Creates URLs using tag UIDs (format: `https://dnd.qrand.me/r/{UID}`)
- 📝 **NDEF Formatting**: Properly formats NFC Forum Type 2 tags
- 🔄 **Batch Processing**: Continuously processes multiple tags
- 🏷️ **UID Integration**: Incorporates tag unique identifiers into written URLs

#### Usage
```bash
cd nfcwriter
go run main.go
```

#### What it does
1. Waits for NFC tag to be placed on reader
2. Reads the tag's unique identifier (UID)
3. Formats the tag as NFC Forum Type 2 format
4. Creates a URL using the UID: `https://dnd.qrand.me/r/{UID}`
5. Writes the URL as NDEF data to the tag
6. Waits for tag removal before processing the next tag

#### Workflow
```
Place tag on reader → Read UID → Format tag → Write URL → Remove tag → Repeat
```

### 3. NFC UID Service (`uid/`)

A background service that reads NFC tag UIDs and automatically copies them to the clipboard with optional auto-paste functionality.

#### Features
- 🔄 **Background Service**: Runs silently as a system service
- 📋 **Clipboard Integration**: Automatically copies UIDs to clipboard
- ⌨️ **Auto-Paste**: Optional automatic paste with Enter key
- 🎯 **Multiple Formats**: Supports hex, reversed hex, and decimal formats
- 🖥️ **Cross-Platform**: Windows, Linux, and macOS support
- 🔌 **Auto-Recovery**: Handles NFC reader disconnections gracefully
- 📝 **Comprehensive Logging**: Debug and info logging options

#### Installation & Usage

#### Windows
```batch
cd uid
install.bat  # Run as Administrator
```

#### Linux
```bash
cd uid
sudo ./install.sh
```

#### macOS
```bash
cd uid
sudo ./install-macos.sh
```

#### Manual Usage
```bash
cd uid
go run main.go                    # Run as service (silent)
go run main.go -debug             # Run with debug output
go run main.go -test              # Test mode - read one card
go run main.go -format hex        # Use hex format
go run main.go -no-paste          # Disable auto-paste
```

#### UID Formats
- **hex**: Standard hexadecimal (e.g., `04A1B2C3`)
- **hex-reversed**: Reversed byte order (e.g., `C3B2A104`)
- **decimal**: Decimal format for short UIDs (e.g., `77654321`)

#### Service Management

**Windows:**
```batch
sc start NFCUIDService
sc stop NFCUIDService
sc query NFCUIDService
```

**Linux:**
```bash
sudo systemctl start nfc-uid-service
sudo systemctl stop nfc-uid-service
sudo journalctl -u nfc-uid-service -f
```

**macOS:**
```bash
sudo launchctl start com.nfc.uid.service
sudo launchctl stop com.nfc.uid.service
```

## System Requirements

### Hardware
- NFC reader supporting PC/SC interface
- NFC tags (NTAG213, NTAG215, NTAG216, or compatible Type 2 tags)

### Software
- **Windows**: PC/SC Smart Card service
- **Linux**: `pcscd` daemon
- **macOS**: PC/SC framework (pre-installed)
- **All platforms**: Go 1.19+ for building from source

### Dependencies
- `github.com/ebfe/scard` - PC/SC interface for NFC communication
- `github.com/atotto/clipboard` - Cross-platform clipboard access (uid service only)

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   nfcreader     │    │   nfcwriter     │    │   uid service   │
│                 │    │                 │    │                 │
│ • Tag analysis  │    │ • Tag writing   │    │ • UID reading   │
│ • NDEF parsing  │    │ • URL encoding  │    │ • Clipboard     │
│ • Memory dumps  │    │ • Batch process │    │ • Auto-paste    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   PC/SC Layer   │
                    │                 │
                    │ • NFC Readers   │
                    │ • Tag Protocol  │
                    └─────────────────┘
```

## Use Cases

### NFC Reader
- **Development**: Analyze NFC tag structure and data
- **Debugging**: Troubleshoot NFC tag formatting issues
- **Research**: Study NFC tag memory layout and NDEF structure
- **Quality Control**: Verify tag formatting and data integrity

### NFC Writer
- **URL Tags**: Create smart tags that link to dynamic content
- **UID-Based Content**: Generate unique URLs based on tag identifiers
- **Batch Programming**: Program multiple tags with unique identifiers
- **IoT Integration**: Write configuration data to NFC tags

### NFC UID Service
- **Access Control**: Copy user IDs from access cards
- **Inventory**: Quickly capture asset tag IDs
- **Authentication**: Streamlined UID entry for login systems
- **Data Entry**: Automated UID input into applications

## Building from Source

### Prerequisites
```bash
# Install Go (version 1.19 or later)
# Download from https://golang.org/dl/

# Install PC/SC dependencies
# Windows: PC/SC service is pre-installed
# Linux: sudo apt-get install pcscd libpcsclite-dev
# macOS: PC/SC framework is pre-installed
```

### Build All Applications
```bash
# Build nfcwriter
cd nfcwriter
go mod download
go build -o nfcwriter main.go

# Build nfcreader
cd ../nfcreader
go mod download
go build -o nfcreader main.go

# Build uid service
cd ../uid
go mod download
go build -o nfc-uid-service main.go
```

## Configuration

### Environment Variables
- None required. All applications auto-detect NFC readers.

### NFC Reader Compatibility
- ACR122U
- ACS ACR1252
- Any PC/SC compatible NFC reader
- Contactless smart card readers

## Troubleshooting

### Common Issues

1. **No NFC reader detected**
   - Ensure reader is connected and drivers are installed
   - Check that PC/SC service is running
   - Try unplugging and reconnecting the reader

2. **Tag not recognized**
   - Verify tag is NFC Forum Type 2 compatible
   - Check tag is not damaged or locked
   - Ensure tag is properly positioned on reader

3. **Permission errors (Linux/macOS)**
   - Ensure user has access to PC/SC devices
   - Try running with `sudo` for testing
   - Check udev rules for device permissions

4. **Service installation fails**
   - Run installation scripts with appropriate privileges
   - Ensure Go dependencies are installed
   - Check that PC/SC service is running

### Debug Mode
Enable verbose logging to troubleshoot issues:

```bash
# NFC Reader
cd nfcreader
go run main.go  # Run with output

# NFC Writer
cd nfcwriter
go run main.go  # Run with output

# UID Service
cd uid
go run main.go -debug  # Enable debug logging
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the BSD-2-Clause License. See individual application directories for specific license information.

## Acknowledgments

- Built with the Go programming language
- Uses PC/SC standard for NFC communication
- Inspired by various open-source NFC projects
- Compatible with NXP NTAG and MIFARE Ultralight tags

## Support

For issues and questions:
1. Check the troubleshooting section above
2. Enable debug mode for detailed logs
3. Create an issue with system information and logs
4. Review the README files in individual application directories

---

**Note**: This suite is designed for development, testing, and integration purposes. Ensure compliance with local regulations when using NFC technology in production environments.
