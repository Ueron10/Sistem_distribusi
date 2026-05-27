@echo off
REM Start multiple distributed counter nodes on Windows
REM This script ensures it runs from the repository root and uses the binary in bin\

echo Starting Distributed Counter System nodes...

REM Change working directory to repo root (scripts\windows -> scripts -> repo root)
pushd "%~dp0\..\.."

REM Path to binary
set "BIN=%CD%\bin\distributed-counter.exe"

REM Build if binary missing
if not exist "%BIN%" (
    echo Binary not found. Building into %BIN%...
    go build -o "%BIN%" .
    if errorlevel 1 (
        echo Build failed. Please check your Go installation.
        popd
        pause
        exit /b 1
    )
)

echo Starting Node 1 on port 8080...
start "Distributed Counter - Node 1" cmd /k ""%BIN%" -id node1 -addr localhost:8080"

timeout /t 2 /nobreak

echo Starting Node 2 on port 8081...
start "Distributed Counter - Node 2" cmd /k ""%BIN%" -id node2 -addr localhost:8081 -peer localhost:8080"

timeout /t 2 /nobreak

echo Starting Node 3 on port 8082...
start "Distributed Counter - Node 3" cmd /k ""%BIN%" -id node3 -addr localhost:8082 -peer localhost:8080"

echo.
echo All nodes started successfully!
echo Each node is running in a separate terminal window.
echo Close the windows to stop the nodes.
echo.
echo You can now run scripts\windows\test.bat to test the system.

REM Return to original directory
popd
