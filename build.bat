@echo off
echo Installing Go dependencies...
go mod tidy
if errorlevel 1 (
    echo Failed to install dependencies
    exit /b 1
)
echo.
echo Building server...
go build -o workout-tracker.exe ./cmd/server
if errorlevel 1 (
    echo Build failed
    exit /b 1
)
echo.
echo Build successful! Run workout-tracker.exe to start the server.
