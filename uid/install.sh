#!/bin/bash
# Linux Service Installation Script for NFC UID Service
# Run with sudo

set -e

SERVICE_NAME="nfc-uid-service"
SERVICE_USER="nfc"
INSTALL_DIR="/opt/nfc-uid-service"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

echo "Installing NFC UID Service for Linux..."

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root (use sudo)" 
   exit 1
fi

# Install dependencies
echo "Installing dependencies..."
if command -v apt-get >/dev/null; then
    # Debian/Ubuntu
    apt-get update
    apt-get install -y pcscd pcsc-tools libpcsclite-dev golang-go xdotool
    systemctl enable pcscd
    systemctl start pcscd
elif command -v yum >/dev/null; then
    # RedHat/CentOS
    yum install -y pcsc-lite pcsc-lite-devel pcsc-tools golang xdotool
    systemctl enable pcscd
    systemctl start pcscd
elif command -v pacman >/dev/null; then
    # Arch Linux
    pacman -S --noconfirm pcsclite ccid go xdotool
    systemctl enable pcscd
    systemctl start pcscd
else
    echo "Unsupported Linux distribution. Please install pcscd, Go, and xdotool manually."
    exit 1
fi

# Create service user
echo "Creating service user..."
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --shell /bin/false --home-dir "$INSTALL_DIR" "$SERVICE_USER"
fi

# Create installation directory
echo "Creating installation directory..."
mkdir -p "$INSTALL_DIR"

# Build the Go application
echo "Building NFC UID Service..."
go build -o "$INSTALL_DIR/${SERVICE_NAME}" .

# Set permissions
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod +x "$INSTALL_DIR/${SERVICE_NAME}"

# Create systemd service file
echo "Creating systemd service..."
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=NFC UID to Clipboard Service
After=network.target pcscd.service
Wants=pcscd.service
Requires=pcscd.service

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/${SERVICE_NAME} -service
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and enable service
echo "Enabling and starting service..."
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

echo ""
echo "NFC UID Service installed successfully!"
echo ""
echo "Service management commands:"
echo "  Start:   sudo systemctl start $SERVICE_NAME"
echo "  Stop:    sudo systemctl stop $SERVICE_NAME"
echo "  Status:  sudo systemctl status $SERVICE_NAME"
echo "  Logs:    sudo journalctl -u $SERVICE_NAME -f"
echo "  Disable: sudo systemctl disable $SERVICE_NAME"
echo ""
echo "The service will automatically start when the system boots."
echo "Check the service status with: sudo systemctl status $SERVICE_NAME"
