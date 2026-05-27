@echo off
REM Build script for Distributed Counter System (Windows)
REM Ensures build runs from repository root and outputs to bin\

echo Building Distributed Counter System...

pushd "%~dp0\..\.."

go build -o bin\distributed-counter.exe .

if errorlevel 1 (
    echo Build failed!
    popd
    pause
    exit /b 1
)

echo Build successful! Binary created: bin\distributed-counter.exe
popd
pause
