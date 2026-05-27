@echo off
REM Test script for Distributed Counter System (Windows)
REM This script demonstrates basic functionality of the system

echo Distributed Counter System Test Script
echo ==================================
echo.

REM Base URLs for the three nodes
set NODE1=http://localhost:8080
set NODE2=http://localhost:8081
set NODE3=http://localhost:8082

REM Check if nodes are running
echo Checking node health...
curl -s %NODE1%/health
curl -s %NODE2%/health
curl -s %NODE3%/health
echo.

REM Test 1: Increment on different nodes
echo Test 1: Increment counter on different nodes
echo Incrementing by 5 at node1...
curl -X POST %NODE1%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 5}"
echo.
echo Incrementing by 3 at node2...
curl -X POST %NODE2%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 3}"
echo.
echo Incrementing by 2 at node3...
curl -X POST %NODE3%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 2}"
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
    curl -X POST %NODE1%/counter/increment -H "Content-Type: application/json" -d "{\"delta\": 1}"
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
