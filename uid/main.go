package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ebfe/scard"
)

// Service configuration
type Config struct {
	ServiceName   string
	ReadInterval  time.Duration
	RetryInterval time.Duration
	MaxRetries    int
	AutoPaste     bool
	UIDFormat     string // "hex", "hex-reversed", "decimal"
	LogLevel      string // "info", "debug", "error"
}

// NFCService represents the background NFC UID service
type NFCService struct {
	config  Config
	ctx     *scard.Context
	reader  string
	running bool
	logger  *log.Logger
}

// Default configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:   "NFCUIDService",
		ReadInterval:  100 * time.Millisecond,
		RetryInterval: 2 * time.Second,
		MaxRetries:    10,
		AutoPaste:     true,
		UIDFormat:     "hex",
		LogLevel:      "info",
	}
}

// NewNFCService creates a new NFC service instance
func NewNFCService(config Config) *NFCService {
	var logger *log.Logger

	// In normal mode, discard all logs; in debug mode, log to stdout
	if config.LogLevel == "debug" {
		logger = log.New(os.Stdout, "[NFCUIDService] ", log.LstdFlags)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	return &NFCService{
		config: config,
		logger: logger,
	}
}

// Initialize sets up the PC/SC context and finds available readers
func (s *NFCService) Initialize() error {
	s.logger.Printf("Initializing %s...", s.config.ServiceName)

	// Establish PC/SC context
	ctx, err := scard.EstablishContext()
	if err != nil {
		return fmt.Errorf("failed to establish PC/SC context: %w", err)
	}
	s.ctx = ctx

	// Find available readers
	if err := s.findReader(); err != nil {
		ctx.Release()
		return fmt.Errorf("failed to find NFC reader: %w", err)
	}

	s.logger.Printf("Successfully initialized with reader: %s", s.reader)
	return nil
}

// findReader discovers available PC/SC readers
func (s *NFCService) findReader() error {
	readers, err := s.ctx.ListReaders()
	if err != nil {
		return fmt.Errorf("failed to list readers: %w", err)
	}

	if len(readers) == 0 {
		return fmt.Errorf("no PC/SC readers found")
	}

	s.reader = readers[0]
	s.logger.Printf("Found %d reader(s), using: %s", len(readers), s.reader)
	return nil
}

// Start begins the background service loop
func (s *NFCService) Start() error {
	if s.ctx == nil {
		return fmt.Errorf("service not initialized")
	}

	s.running = true

	s.logger.Printf("Starting %s in background mode...", s.config.ServiceName)
	s.logger.Printf("Configuration: AutoPaste=%v, Format=%s", s.config.AutoPaste, s.config.UIDFormat)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		s.logger.Printf("Received shutdown signal, stopping service...")
		s.Stop()
	}()

	// Main service loop
	for s.running {
		if err := s.processCardCycle(); err != nil {
			s.logger.Printf("Card processing error: %v", err)

			// Try to recover by reinitializing reader connection
			if err := s.recoverReader(); err != nil {
				s.logger.Printf("Reader recovery failed: %v", err)
				time.Sleep(s.config.RetryInterval)
			}
		}

		time.Sleep(s.config.ReadInterval)
	}

	return nil
}

// Stop gracefully shuts down the service
func (s *NFCService) Stop() {
	s.running = false
	if s.ctx != nil {
		s.ctx.Release()
	}
	s.logger.Printf("Service stopped")
}

// processCardCycle handles one complete card detection and processing cycle
func (s *NFCService) processCardCycle() error {
	// Wait for card presence
	if !s.waitForCardPresent(5 * time.Second) {
		return nil // Timeout, continue loop
	}

	// Connect to card
	card, err := s.connectToCard()
	if err != nil {
		s.waitForCardRemoval(1 * time.Second) // Brief wait before continuing
		return fmt.Errorf("failed to connect to card: %w", err)
	}
	defer card.Disconnect(scard.LeaveCard)

	// Read UID
	uid, err := s.getUID(card)
	if err != nil {
		return fmt.Errorf("failed to read UID: %w", err)
	}

	// Process UID
	if err := s.processUID(uid); err != nil {
		return fmt.Errorf("failed to process UID: %w", err)
	}

	// Wait for card removal to avoid re-processing
	s.waitForCardRemoval(10 * time.Second)

	return nil
}

// connectToCard establishes connection to the card with retries
func (s *NFCService) connectToCard() (*scard.Card, error) {
	var card *scard.Card
	var err error

	for i := 0; i < s.config.MaxRetries; i++ {
		card, err = s.ctx.Connect(s.reader, scard.ShareShared, scard.ProtocolAny)
		if err == nil {
			return card, nil
		}
		time.Sleep(s.config.ReadInterval)
	}

	return nil, err
}

// getUID reads the UID from the connected card
func (s *NFCService) getUID(card *scard.Card) ([]byte, error) {
	// Use the ACR/PCSC pseudo-APDU FF CA 00 00 00 to fetch UID
	resp, err := card.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
	if err != nil {
		return nil, err
	}

	if len(resp) < 2 {
		return nil, fmt.Errorf("short APDU response")
	}

	// Check status words
	sw1 := resp[len(resp)-2]
	sw2 := resp[len(resp)-1]
	if sw1 != 0x90 || sw2 != 0x00 {
		return nil, fmt.Errorf("APDU failed: SW=%02X%02X", sw1, sw2)
	}

	return resp[:len(resp)-2], nil
}

// processUID handles the UID formatting, clipboard copy, and paste operations
func (s *NFCService) processUID(uid []byte) error {
	if len(uid) == 0 {
		return fmt.Errorf("empty UID")
	}

	// Format UID according to configuration
	formattedUID := s.formatUID(uid)

	s.logger.Printf("Detected NFC UID: %s", formattedUID)

	// Copy to clipboard
	if err := clipboard.WriteAll(formattedUID); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	s.logger.Printf("Copied UID to clipboard: %s", formattedUID)

	// Auto-paste if enabled (default behavior)
	if s.config.AutoPaste {
		if err := s.performPaste(); err != nil {
			s.logger.Printf("Auto-paste failed: %v", err)
			// Don't return error here, clipboard copy was successful
		} else {
			s.logger.Printf("Auto-pasted UID and pressed Enter")
		}
	}

	return nil
}

// formatUID converts the raw UID bytes to the specified format
func (s *NFCService) formatUID(uid []byte) string {
	switch s.config.UIDFormat {
	case "hex":
		return strings.ToUpper(hex.EncodeToString(uid))
	case "hex-reversed":
		// Reverse the byte order
		reversed := make([]byte, len(uid))
		for i, j := 0, len(uid)-1; i < len(uid); i, j = i+1, j-1 {
			reversed[i] = uid[j]
		}
		return strings.ToUpper(hex.EncodeToString(reversed))
	case "decimal":
		// Convert to decimal (for shorter UIDs)
		if len(uid) <= 4 {
			var val uint32
			for i, b := range uid {
				val |= uint32(b) << (8 * (len(uid) - 1 - i))
			}
			return fmt.Sprintf("%d", val)
		}
		// For longer UIDs, fall back to hex
		return strings.ToUpper(hex.EncodeToString(uid))
	default:
		return strings.ToUpper(hex.EncodeToString(uid))
	}
}

// performPaste simulates Ctrl+V keypress to paste the clipboard content, then presses Enter
func (s *NFCService) performPaste() error {
	// Small delay to ensure the target application is ready
	time.Sleep(50 * time.Millisecond)

	var pasteCmd *exec.Cmd
	var enterCmd *exec.Cmd

	// Use system commands to simulate keypress based on operating system
	switch runtime.GOOS {
	case "windows":
		// Use PowerShell to send Ctrl+V then Enter
		pasteCmd = exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait('^v')")
		enterCmd = exec.Command("powershell", "-Command", "Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait('{ENTER}')")
	case "linux":
		// Try xdotool first, fall back to xclip
		if _, err := exec.LookPath("xdotool"); err == nil {
			pasteCmd = exec.Command("xdotool", "key", "ctrl+v")
			enterCmd = exec.Command("xdotool", "key", "Return")
		} else if _, err := exec.LookPath("xte"); err == nil {
			pasteCmd = exec.Command("xte", "keydown Control_L", "key v", "keyup Control_L")
			enterCmd = exec.Command("xte", "key Return")
		} else {
			return fmt.Errorf("no suitable keyboard automation tool found (install xdotool or xautomation)")
		}
	case "darwin": // macOS
		// Use AppleScript to send Cmd+V then Enter
		pasteScript := `tell application "System Events" to keystroke "v" using command down`
		enterScript := `tell application "System Events" to keystroke return`
		pasteCmd = exec.Command("osascript", "-e", pasteScript)
		enterCmd = exec.Command("osascript", "-e", enterScript)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Execute paste command
	if pasteCmd != nil {
		if err := pasteCmd.Run(); err != nil {
			return fmt.Errorf("failed to paste: %w", err)
		}
	} else {
		return fmt.Errorf("failed to create paste command")
	}

	// Small delay between paste and enter
	time.Sleep(50 * time.Millisecond)

	// Execute enter command
	if enterCmd != nil {
		if err := enterCmd.Run(); err != nil {
			return fmt.Errorf("failed to press enter: %w", err)
		}
	} else {
		return fmt.Errorf("failed to create enter command")
	}

	return nil
}

// waitForCardPresent blocks until a card is detected or timeout occurs
func (s *NFCService) waitForCardPresent(timeout time.Duration) bool {
	rs := []scard.ReaderState{{Reader: s.reader, CurrentState: scard.StateUnaware}}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && s.running {
		err := s.ctx.GetStatusChange(rs, 500*time.Millisecond)
		if err != nil {
			continue
		}

		st := rs[0].EventState
		rs[0].CurrentState = st

		if st&scard.StatePresent != 0 {
			return true
		}
	}

	return false
}

// waitForCardRemoval blocks until the card is removed or timeout occurs
func (s *NFCService) waitForCardRemoval(timeout time.Duration) bool {
	rs := []scard.ReaderState{{Reader: s.reader, CurrentState: scard.StateUnaware}}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && s.running {
		err := s.ctx.GetStatusChange(rs, 500*time.Millisecond)
		if err != nil {
			continue
		}

		st := rs[0].EventState
		rs[0].CurrentState = st

		if st&scard.StatePresent == 0 {
			return true
		}
	}

	return false
}

// recoverReader attempts to recover from reader disconnection
func (s *NFCService) recoverReader() error {
	s.logger.Printf("Attempting to recover reader connection...")

	// Release current context
	if s.ctx != nil {
		s.ctx.Release()
	}

	// Re-establish context
	ctx, err := scard.EstablishContext()
	if err != nil {
		return fmt.Errorf("failed to re-establish PC/SC context: %w", err)
	}
	s.ctx = ctx

	// Re-discover readers
	if err := s.findReader(); err != nil {
		return fmt.Errorf("failed to rediscover readers: %w", err)
	}

	s.logger.Printf("Successfully recovered reader connection: %s", s.reader)
	return nil
}

// printUsage displays command line usage information
func printUsage() {
	fmt.Printf(`NFC UID to Clipboard Service

Usage: %s [options]

Options:
  -h, --help           Show this help message
  -format string       UID format: hex, hex-reversed, decimal (default: hex)
  -no-paste           Disable automatic paste+enter functionality
  -service            Run as background service (default)
  -debug              Enable debug logging
  -test               Test mode - read one card and exit

Examples:
  %s                           # Run as service with default settings
  %s -format hex-reversed      # Use reversed hex format
  %s -no-paste                 # Only copy to clipboard, don't auto-paste+enter
  %s -test                     # Test mode - read one card and exit

Service Installation:
  Windows: Run as Administrator and the service will auto-install
  Linux:   Use systemctl to manage the service
  macOS:   Use launchctl to manage the service

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	// Check for help flag first, before any other processing
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
			os.Exit(0)
		}
	}

	// Parse command line arguments
	config := DefaultConfig()
	testMode := false

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-format":
			if i+1 < len(os.Args) {
				config.UIDFormat = os.Args[i+1]
				i++ // Skip next argument as it's the format value
			}
		case "-no-paste":
			config.AutoPaste = false
		case "-debug":
			config.LogLevel = "debug"
		case "-test":
			testMode = true
		}
	}

	// Validate format
	if config.UIDFormat != "hex" && config.UIDFormat != "hex-reversed" && config.UIDFormat != "decimal" {
		fmt.Printf("Invalid format: %s. Use: hex, hex-reversed, or decimal\n", config.UIDFormat)
		os.Exit(1)
	}

	// Create and initialize service
	service := NewNFCService(config)
	if err := service.Initialize(); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}
	defer service.Stop()

	if testMode {
		// Test mode - read one card and exit
		service.logger.Printf("Running in test mode - will read one card and exit")

		if err := service.processCardCycle(); err != nil {
			log.Fatalf("Test failed: %v", err)
		}

		service.logger.Printf("Test completed successfully")
		return
	}

	// Normal service mode - startup messages only in debug
	service.logger.Printf("NFC UID Service starting...")
	service.logger.Printf("Place NFC tags on the reader to copy UIDs to clipboard")

	if err := service.Start(); err != nil {
		log.Fatalf("Service failed: %v", err)
	}
}
