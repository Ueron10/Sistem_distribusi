@echo off
REM Test script for Distributed Counter System (Windows)
REM This script demonstrates basic functionality of the system

echo Distributed Counter System Test Script
echo ==================================
echo.

REM Ensure we run from repository root
pushd "%~dp0\..\.."

REM Base URLs for the three nodes
set NODE1=http://localhost:8080
set NODE2=http://localhost:8081
set NODE3=http://localhost:8082

REM Health readiness helper
set MAX_RETRIES=15
set RETRY_DELAY=2

echo Checking node health and readiness...
call :wait_for_node %NODE1% %MAX_RETRIES%
call :wait_for_node %NODE2% %MAX_RETRIES%
call :wait_for_node %NODE3% %MAX_RETRIES%
echo.

REM Test 1: Increment on different nodes
echo Test 1: Increment counter on different nodes
echo Incrementing by 5 at node1...
curl -s -X POST %NODE1%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 5}"
echo.
echo Incrementing by 3 at node2...
curl -s -X POST %NODE2%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 3}"
echo.
echo Incrementing by 2 at node3...
curl -s -X POST %NODE3%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 2}"
echo.

REM Wait for gossip to propagate
echo Waiting for gossip propagation (5 seconds)...
timeout /t 5 /nobreak
echo.

REM Test 2: Check values on all nodes
echo Test 2: Check counter values on all nodes
echo Value at node1:
curl -s %NODE1%/counter/value
echo.
echo Value at node2:
curl -s %NODE2%/counter/value
echo.
echo Value at node3:
curl -s %NODE3%/counter/value
echo.

REM Test 3: List peers
echo Test 3: List peers at node1
curl -s %NODE1%/peers
echo.

REM Test 4: Increment multiple times
echo Test 4: Increment multiple times on single node
for /L %%i in (1,1,3) do (
    echo Increment %%i...
    curl -s -X POST %NODE1%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 1}"
    echo.
    timeout /t 1 /nobreak
)
echo.

REM Wait for gossip
echo Waiting for gossip propagation (5 seconds)...
timeout /t 5 /nobreak
echo.

REM Final value check
echo Test 5: Final counter values
echo Value at node1:
curl -s %NODE1%/counter/value
echo.
echo Value at node2:
curl -s %NODE2%/counter/value
echo.
echo Value at node3:
curl -s %NODE3%/counter/value
echo.

echo Test completed!
pause

popd

goto :eof

:wait_for_node
REM args: %1 = node url, %2 = max retries
set NODEURL=%1
set RETRIES=%2
set /a i=0
for /f "tokens=* delims=" %%S in ('cmd /c echo checking') do set dummy=%%S
:wait_loop
for /f "tokens=* delims=" %%S in ('curl -s -o NUL -w "%%{http_code}" %NODEURL%/health') do set CODE=%%S
if "%CODE%"=="200" (
    echo %NODEURL% is healthy.
    goto :eof
)
set /a i+=1
if %i% GEQ %RETRIES% (
    echo %NODEURL% did not become healthy after %RETRIES% tries.
    goto :eof
)
echo Waiting for %NODEURL% (%i%/%RETRIES%)...
timeout /t %RETRY_DELAY% /nobreak
goto wait_loop
