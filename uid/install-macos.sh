#!/bin/bash
# macOS LaunchDaemon Installation Script for NFC UID Service

set -e

SERVICE_NAME="com.nfc.uid.service"
INSTALL_DIR="/usr/local/bin"
PLIST_DIR="/Library/LaunchDaemons"
PLIST_FILE="$PLIST_DIR/${SERVICE_NAME}.plist"
BINARY_NAME="nfc-uid-service"

echo "Installing NFC UID Service for macOS..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Check for Homebrew and install dependencies
echo "Checking dependencies..."
if ! command -v brew >/dev/null; then
    echo "Homebrew not found. Please install Homebrew first:"
    echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    exit 1
fi

# Install dependencies via Homebrew
echo "Installing dependencies via Homebrew..."
sudo -u $(logname) brew install pcsc-lite go

# Build the Go application
echo "Building NFC UID Service..."
go build -o "${INSTALL_DIR}/${BINARY_NAME}" .

# Set permissions
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# Create LaunchDaemon plist file
echo "Creating LaunchDaemon configuration..."
cat > "$PLIST_FILE" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BINARY_NAME}</string>
        <string>-service</string>
    </array>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    
    <key>WorkingDirectory</key>
    <string>/tmp</string>
    
    <key>StandardOutputPath</key>
    <string>/var/log/nfc-uid-service.log</string>
    
    <key>StandardErrorPath</key>
    <string>/var/log/nfc-uid-service.error.log</string>
    
    <key>ThrottleInterval</key>
    <integer>5</integer>
</dict>
</plist>
EOF

# Set proper permissions for plist file
chown root:wheel "$PLIST_FILE"
chmod 644 "$PLIST_FILE"

# Load and start the service
echo "Loading and starting service..."
launchctl load "$PLIST_FILE"
launchctl start "$SERVICE_NAME"

echo ""
echo "NFC UID Service installed successfully!"
echo ""
echo "Service management commands:"
echo "  Start:   sudo launchctl start $SERVICE_NAME"
echo "  Stop:    sudo launchctl stop $SERVICE_NAME"
echo "  Status:  sudo launchctl list | grep $SERVICE_NAME"
echo "  Logs:    tail -f /var/log/nfc-uid-service.log"
echo "  Unload:  sudo launchctl unload $PLIST_FILE"
echo ""
echo "The service will automatically start when macOS boots."
echo "Check the service status with: sudo launchctl list | grep $SERVICE_NAME"

# Note about permissions
echo ""
echo "IMPORTANT: macOS Security Notes:"
echo "1. You may need to grant accessibility permissions to the terminal"
echo "2. Go to System Preferences > Security & Privacy > Privacy > Accessibility"
echo "3. Add Terminal.app or your terminal emulator to the allowed apps"
echo "4. This is required for the auto-paste functionality to work"
