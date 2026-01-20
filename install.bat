@echo off
echo ==========================================
echo   Syncase Silent App Installer
echo ==========================================

REM Check for admin rights
NET SESSION >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
    echo This script requires administrator privileges.
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo 1. Building application...
go build -o bin/syncase.exe

echo.
echo 2. Installing as Windows Service...
sc create SyncaseSilentService binPath="%~dp0bin\syncase.exe" start=auto
sc description SyncaseSilentService "Syncase Silent File Synchronization Service"
sc start SyncaseSilentService

echo.
echo 3. Setting up configuration...
if not exist config.json (
    echo Creating default config.json...
    echo { > config.json
    echo   "watchedFolder": "C:/Users/%%USERNAME%%/Documents/Watched_folder", >> config.json
    echo   "rclone_remote": "BalalaGdrive", >> config.json
    echo   "encryption_key": "" >> config.json
    echo } >> config.json
    echo.
    echo Please edit config.json to set your encryption key and rclone remote!
)

echo.
echo ==========================================
echo   Installation Complete!
echo ==========================================
echo.
echo The service is now running silently in the background.
echo.
echo To check status: sc query SyncaseSilentService
echo To stop: sc stop SyncaseSilentService
echo To uninstall: uninstall.bat
echo.
pause