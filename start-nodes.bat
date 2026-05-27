@echo off
REM Start multiple distributed counter nodes on Windows
REM Open new terminal windows for each node

echo Starting Distributed Counter System nodes...
echo.

REM Check if the binary exists
if not exist "distributed-counter.exe" (
    echo Binary not found. Building...
    go build -o distributed-counter.exe
    if errorlevel 1 (
        echo Build failed. Please check your Go installation.
        pause
        exit /b 1
    )
)

echo Starting Node 1 on port 8080...
start "Distributed Counter - Node 1" cmd /k "distributed-counter.exe -id node1 -addr localhost:8080"

timeout /t 2 /nobreak

echo Starting Node 2 on port 8081...
start "Distributed Counter - Node 2" cmd /k "distributed-counter.exe -id node2 -addr localhost:8081 -peer localhost:8080"

timeout /t 2 /nobreak

echo Starting Node 3 on port 8082...
start "Distributed Counter - Node 3" cmd /k "distributed-counter.exe -id node3 -addr localhost:8082 -peer localhost:8080"

echo.
echo All nodes started successfully!
echo Each node is running in a separate terminal window.
echo Close the windows to stop the nodes.
echo.
echo You can now run test.bat to test the system.
