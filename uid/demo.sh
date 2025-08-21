#!/bin/bash
# Demo script for NFC UID Service

echo "🔥 NFC UID to Clipboard Service Demo"
echo "===================================="
echo ""

# Check if binary exists
if [ ! -f "./nfc-uid-service" ]; then
    echo "Building application..."
    make build
    echo ""
fi

echo "📋 Available Commands:"
echo ""
echo "1. Silent Operation (recommended):"
echo "   ./nfc-uid-service"
echo "   → Runs completely silently, pastes UIDs + Enter automatically"
echo ""
echo "2. Debug Mode:"
echo "   ./nfc-uid-service -debug"
echo "   → Shows all operations for troubleshooting"
echo ""
echo "3. Test Mode:"
echo "   ./nfc-uid-service -test"
echo "   → Read one card and exit (with output)"
echo ""
echo "4. Format Options:"
echo "   ./nfc-uid-service -format hex-reversed"
echo "   ./nfc-uid-service -format decimal"
echo ""
echo "5. Clipboard Only (no auto-paste+enter):"
echo "   ./nfc-uid-service -no-paste"
echo ""

read -p "🚀 Run demo in debug mode? (y/N): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "Starting NFC UID Service in debug mode..."
    echo "Place an NFC tag on your reader to see it in action!"
    echo "Press Ctrl+C to stop"
    echo ""
    ./nfc-uid-service -debug
else
    echo ""
    echo "Demo completed. To run the service:"
    echo "  ./nfc-uid-service         # Silent operation"
    echo "  ./nfc-uid-service -debug  # With debug output"
    echo ""
fi
