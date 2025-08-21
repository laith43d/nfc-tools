#!/bin/bash
# macOS LaunchDaemon Uninstallation Script for NFC UID Service

set -e

SERVICE_NAME="com.nfc.uid.service"
INSTALL_DIR="/usr/local/bin"
PLIST_DIR="/Library/LaunchDaemons"
PLIST_FILE="$PLIST_DIR/${SERVICE_NAME}.plist"
BINARY_NAME="nfc-uid-service"

echo "Uninstalling NFC UID Service for macOS..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Stop and unload the service
if launchctl list | grep -q "$SERVICE_NAME"; then
    echo "Stopping and unloading service..."
    launchctl stop "$SERVICE_NAME" 2>/dev/null || true
    launchctl unload "$PLIST_FILE" 2>/dev/null || true
fi

# Remove plist file
if [[ -f "$PLIST_FILE" ]]; then
    echo "Removing LaunchDaemon configuration..."
    rm -f "$PLIST_FILE"
fi

# Remove binary
if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
    echo "Removing executable..."
    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
fi

# Remove log files
echo "Removing log files..."
rm -f /var/log/nfc-uid-service.log
rm -f /var/log/nfc-uid-service.error.log

echo ""
echo "NFC UID Service has been completely uninstalled."
echo ""
echo "Note: PC/SC libraries were not removed as they may be used by other applications."
echo "If you want to remove them as well, run:"
echo "  brew uninstall pcsc-lite"
