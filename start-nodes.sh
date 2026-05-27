#!/bin/bash

# Start multiple distributed counter nodes on Unix/Linux
# Open new terminal windows for each node

echo "Starting Distributed Counter System nodes..."
echo ""

# Check if the binary exists
if [ ! -f "distributed-counter" ]; then
    echo "Binary not found. Building..."
    go build -o distributed-counter
    if [ $? -ne 0 ]; then
        echo "Build failed. Please check your Go installation."
        exit 1
    fi
fi

# Function to start a node in a new terminal
start_node() {
    local title=$1
    local id=$2
    local addr=$3
    local peer=$4
    
    if command -v gnome-terminal &> /dev/null; then
        gnome-terminal --title="$title" -- bash -c "./distributed-counter -id $id -addr $addr -peer $peer; exec bash"
    elif command -v xterm &> /dev/null; then
        xterm -title "$title" -e "./distributed-counter -id $id -addr $addr -peer $peer" &
    elif command -v terminal &> /dev/null; then
        terminal -t "$title" -e "./distributed-counter -id $id -addr $addr -peer $peer" &
    else
        echo "No supported terminal emulator found. Starting nodes in background..."
        ./distributed-counter -id $id -addr $addr -peer $peer &
    fi
}

echo "Starting Node 1 on port 8080..."
start_node "Node 1" "node1" "localhost:8080" ""

sleep 2

echo "Starting Node 2 on port 8081..."
start_node "Node 2" "node2" "localhost:8081" "localhost:8080"

sleep 2

echo "Starting Node 3 on port 8082..."
start_node "Node 3" "node3" "localhost:8082" "localhost:8080"

echo ""
echo "All nodes started successfully!"
echo "Each node is running in a separate terminal window."
echo "Close the windows to stop the nodes."
echo ""
echo "You can now run ./test.sh to test the system."
