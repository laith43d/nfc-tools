#!/bin/bash
# Linux Service Uninstallation Script for NFC UID Service
# Run with sudo

set -e

SERVICE_NAME="nfc-uid-service"
SERVICE_USER="nfc"
INSTALL_DIR="/opt/nfc-uid-service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

echo "Uninstalling NFC UID Service for Linux..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Stop and disable service
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Stopping service..."
    systemctl stop "$SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    echo "Disabling service..."
    systemctl disable "$SERVICE_NAME"
fi

# Remove service file
if [[ -f "$SERVICE_FILE" ]]; then
    echo "Removing service file..."
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
fi

# Remove installation directory
if [[ -d "$INSTALL_DIR" ]]; then
    echo "Removing installation directory..."
    rm -rf "$INSTALL_DIR"
fi

# Remove service user
if id "$SERVICE_USER" &>/dev/null; then
    echo "Removing service user..."
    userdel "$SERVICE_USER"
fi

echo ""
echo "NFC UID Service has been completely uninstalled."
echo ""
echo "Note: PC/SC daemon (pcscd) was not removed as it may be used by other applications."
echo "If you want to remove it as well, run:"
echo "  Debian/Ubuntu: sudo apt-get remove pcscd pcsc-tools"
echo "  RedHat/CentOS: sudo yum remove pcsc-lite pcsc-tools"
echo "  Arch Linux:    sudo pacman -R pcsclite ccid"
