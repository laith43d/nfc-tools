@echo off
REM Windows Service Installation Script for NFC UID Service
REM Run as Administrator

echo Installing NFC UID Service for Windows...

REM Check if running as administrator
net session >nul 2>&1
if %errorLevel% == 0 (
    echo Running as Administrator - OK
) else (
    echo This script must be run as Administrator!
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

REM Build the Go application
echo Building NFC UID Service...
go build -o nfc-uid-service.exe .
if %errorLevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

REM Create service using sc command
echo Creating Windows service...
sc create "NFCUIDService" ^
    binPath= "%~dp0nfc-uid-service.exe -service" ^
    DisplayName= "NFC UID to Clipboard Service" ^
    start= auto ^
    depend= "PCSC"

if %errorLevel% == 0 (
    echo Service created successfully!
    echo Starting service...
    sc start "NFCUIDService"
    
    echo.
    echo Service installed and started successfully!
    echo The service will automatically start when Windows boots.
    echo.
    echo To manage the service:
    echo   Start:   sc start NFCUIDService
    echo   Stop:    sc stop NFCUIDService
    echo   Delete:  sc delete NFCUIDService
    echo.
    echo Service logs will appear in Windows Event Viewer under Application logs.
) else (
    echo Failed to create service!
    echo Make sure you're running as Administrator and try again.
)

pause
