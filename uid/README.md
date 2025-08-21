# NFC UID to Clipboard Service

A Go-based background service that automatically reads NFC tag UIDs and copies them to the clipboard with optional auto-paste functionality. This is a modern replacement for [UIDtoKeyboard](https://github.com/tithanayut/UIDtoKeyboard) with enhanced features and cross-platform support.

## Features

- 🔄 **Silent Background Service**: Runs completely silently as a system service
- 📋 **Clipboard Integration**: Automatically copies UIDs to clipboard
- ⌨️ **Auto-Paste + Enter**: Automatic pasting with Ctrl+V/Cmd+V followed by Enter
- 🔌 **Reader Recovery**: Automatic recovery from NFC reader disconnections
- 🎯 **Multiple Formats**: Support for hex, reversed hex, and decimal formats
- 🖥️ **Cross-Platform**: Windows, Linux, and macOS support
- 📝 **Comprehensive Logging**: Detailed logging for troubleshooting
- 🚀 **Easy Installation**: One-click installation scripts

## Workflow

1. **Silent Background Operation**: Runs completely silently as a system service
2. **NFC Detection**: Continuously monitors for NFC tags with no output
3. **UID Reading**: Reads UID when tag is detected
4. **Clipboard Copy**: Automatically copies UID to clipboard
5. **Auto-Paste + Enter**: Immediately pastes UID at cursor position and presses Enter
6. **Recovery**: Handles reader disconnections gracefully

## Installation

### Prerequisites

- **Windows**: PC/SC Smart Card service enabled
- **Linux**: `pcscd` daemon installed and running
- **macOS**: PC/SC framework (usually pre-installed)
- **Go**: Version 1.19 or later for building from source

### Quick Install

#### Windows
```batch
# Run as Administrator
install.bat
```

#### Linux
```bash
# Run with sudo
sudo ./install.sh
```

#### macOS
```bash
# Run with sudo
sudo ./install-macos.sh
```

## Manual Installation

### 1. Install Dependencies

```bash
# Download dependencies
go mod download
```

### 2. Build Application

```bash
# Build for current platform
go build -o nfc-uid-service .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o nfc-uid-service.exe .

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o nfc-uid-service .
```

### 3. Test Installation

```bash
# Test mode - reads one card and exits
./nfc-uid-service -test

# Manual mode with debug output
./nfc-uid-service -debug
```

## Usage

### Command Line Options

```bash
# Run as service (default - completely silent)
./nfc-uid-service

# Specify UID format (silent operation)
./nfc-uid-service -format hex-reversed

# Disable auto-paste+enter functionality (clipboard copy only)
./nfc-uid-service -no-paste

# Enable debug logging (shows all operations)
./nfc-uid-service -debug

# Test mode (read one card and exit with output)
./nfc-uid-service -test

# Show help
./nfc-uid-service -help
```

### UID Formats

- **hex**: Standard hexadecimal (e.g., `04A1B2C3`)
- **hex-reversed**: Reversed byte order (e.g., `C3B2A104`)
- **decimal**: Decimal format for 4-byte UIDs (e.g., `77654321`)

## Service Management

### Windows

```batch
# Start service
sc start NFCUIDService

# Stop service
sc stop NFCUIDService

# Check status
sc query NFCUIDService

# Remove service
sc delete NFCUIDService
```

### Linux (systemd)

```bash
# Start service
sudo systemctl start nfc-uid-service

# Stop service
sudo systemctl stop nfc-uid-service

# Check status
sudo systemctl status nfc-uid-service

# View logs
sudo journalctl -u nfc-uid-service -f

# Disable auto-start
sudo systemctl disable nfc-uid-service
```

### macOS (launchctl)

```bash
# Start service
sudo launchctl start com.nfc.uid.service

# Stop service
sudo launchctl stop com.nfc.uid.service

# Check status
sudo launchctl list | grep com.nfc.uid.service

# View logs
tail -f /var/log/nfc-uid-service.log

# Unload service
sudo launchctl unload /Library/LaunchDaemons/com.nfc.uid.service.plist
```

## Configuration

The service automatically detects and uses the first available NFC reader. Configuration is done through command-line arguments:

```bash
# Example configurations
./nfc-uid-service -format hex                  # Hex format, silent operation
./nfc-uid-service -format decimal -no-paste    # Decimal format, clipboard only
./nfc-uid-service -debug                       # Show all debug output
```

## Troubleshooting

### Common Issues

1. **No NFC reader detected**
   - Ensure NFC reader is connected and drivers are installed
   - Check that PC/SC service is running
   - Try unplugging and reconnecting the reader

2. **Auto-paste not working**
   - **Windows**: Ensure the service has proper permissions
   - **Linux**: Install `xdotool` or similar X11 automation tools
   - **macOS**: Grant accessibility permissions to the terminal

3. **Service won't start**
   - Check that no other application is using the NFC reader
   - Verify PC/SC dependencies are installed
   - Check service logs for specific error messages

### Debug Mode

Run in debug mode to see detailed operation logs:

```bash
./nfc-uid-service -debug
```

### Test Mode

Test the NFC reading functionality:

```bash
./nfc-uid-service -test
```

## Uninstallation

### Windows
```batch
# Run as Administrator
uninstall.bat
```

### Linux
```bash
# Run with sudo
sudo ./uninstall.sh
```

### macOS
```bash
# Run with sudo
sudo ./uninstall-macos.sh
```

## Comparison with UIDtoKeyboard

| Feature | UIDtoKeyboard | NFC UID Service |
|---------|---------------|-----------------|
| Platform | Windows only | Windows, Linux, macOS |
| Mode | Desktop GUI | Background service |
| Auto-start | Manual setup | System service |
| Recovery | Manual restart | Automatic |
| Formats | 4 formats | 3 optimized formats |
| Logging | Basic | Comprehensive |
| Dependencies | .NET Framework | Self-contained |

## Development

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd uid

# Install dependencies
go mod download

# Build
go build .

# Run tests
go test ./...
```

### Dependencies

- `github.com/ebfe/scard`: PC/SC interface for NFC communication
- `github.com/atotto/clipboard`: Cross-platform clipboard access
- System keyboard automation tools:
  - **Windows**: PowerShell (built-in)
  - **Linux**: xdotool or xautomation
  - **macOS**: AppleScript (built-in)

## License

This project is licensed under the BSD-2-Clause License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## Support

For issues and support:

1. Check the troubleshooting section
2. Enable debug mode for detailed logs
3. Create an issue with logs and system information

## Acknowledgments

- Inspired by [UIDtoKeyboard](https://github.com/tithanayut/UIDtoKeyboard) by Thanayut T.
- Built with the Go programming language
- Uses PC/SC for NFC communication
