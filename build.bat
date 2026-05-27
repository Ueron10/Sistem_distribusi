@echo off
REM Build script for Distributed Counter System (Windows)

echo Building Distributed Counter System...
go build -o distributed-counter.exe

if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful! Binary created: distributed-counter.exe
pause
