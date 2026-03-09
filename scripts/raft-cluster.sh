#!/bin/bash

#
# SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
#

# Define the number of instances you want to start
NUM_INSTANCES=4

# Define the base port number
BASE_PORT=4001

# Define the base directory
BASE_DIR="data"

# Boolean flag to recreate directories
RECREATE_DIRECTORIES=false

# Array to store background process IDs
BG_PIDS=()

# Function to check if a port is available and kill the process using it
check_and_kill_port() {
    local port="$1"
    local pid

    pid=$(lsof -ti :$port)
    if [ -n "$pid" ]; then
        echo "Port $port is already in use. Killing process $pid..."
        kill -9 "$pid"
        sleep 2 # Wait for the process to be killed
    fi
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --new)
            RECREATE_DIRECTORIES=true
            shift
            ;;
        *)
            # Unknown option
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Function to clean up and exit
cleanup_and_exit() {
    echo "Cleaning up and exiting..."

    # Kill all background processes
    for pid in "${BG_PIDS[@]}"; do
        kill "$pid"
    done

    exit 0
}

# Trap exit signals
trap cleanup_and_exit EXIT SIGINT SIGTERM

if [ "$RECREATE_DIRECTORIES" = true ] && [ -d "$BASE_DIR" ]; then
    echo "Removing existing data state and recreating cluster.."
    rm -rf "$BASE_DIR"
fi

# Loop to start instances
for ((i=1; i<=$NUM_INSTANCES; i++)); do
    # Calculate the port for each instance
    PORT=$((BASE_PORT + i-1))

    # Set common environment variables for each instance
    export PORT=$PORT
    export ENV="dev"
    export RAFT_REMOVE_NODE="false"

    # Check if the port is available, and kill the process if it's not
    check_and_kill_port "$PORT"

    echo "Creating instance $i on  $PORT.."
    # Set additional environment variable for the second instance
    if [ $i -ne 1 ]; then
        export RAFT_CLUSTER_ADDRESS="localhost:${BASE_PORT}"
    fi

    # Create or recreate the directory
    DIR_PATH="$BASE_DIR"

    if [ ! -d "$DIR_PATH" ]; then
        mkdir -p "$DIR_PATH"
        chmod +rw "$DIR_PATH"
        echo "Directory created: $DIR_PATH"
    else
        echo "Directory already exists: $DIR_PATH"
    fi

    export RAFT_DATA_DIRECTORY=$DIR_PATH

    # Start the Golang service in the background
    go run cmd/metadata/main.go &
    BG_PIDS+=($!)

    echo "Waiting for 10s before starting the next instance"
    sleep 10
    # Store the endpoint in an array
    ENDPOINTS[$i]="localhost:$PORT"
done

# Output the list of endpoints
echo "List of raft nodes:"
for endpoint in "${ENDPOINTS[@]}"; do
    echo $endpoint
done

# Wait for user input to exit the script
read -rp "Press Enter to exit..."
cleanup_and_exit
