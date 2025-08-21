@echo off
REM Windows Service Uninstallation Script for NFC UID Service
REM Run as Administrator

echo Uninstalling NFC UID Service for Windows...

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

REM Stop the service if running
echo Stopping NFC UID Service...
sc stop "NFCUIDService"

REM Wait a moment for service to stop
timeout /t 3 /nobreak >nul

REM Delete the service
echo Removing Windows service...
sc delete "NFCUIDService"

if %errorLevel% == 0 (
    echo Service removed successfully!
    
    REM Remove the executable
    if exist "nfc-uid-service.exe" (
        echo Removing executable...
        del "nfc-uid-service.exe"
    )
    
    echo.
    echo NFC UID Service has been completely uninstalled.
) else (
    echo Failed to remove service!
    echo The service may not be installed or you need Administrator privileges.
)

pause
