#!/bin/bash

# Check if port number is provided
if [ $# -eq 0 ]; then
    read -p "Enter the port number to monitor: " port
else
    port=$1
fi

# Function to get PID of process using the specified port
get_pid() {
    lsof -i :$port -t
}

# Function to get memory usage of a specific PID in MB
get_process_memory() {
    pid=$1
    if [ -n "$pid" ]; then
        ps -o rss= -p $pid | awk '{print int($1/1024)}'  # Convert KB to MB and round to integer
    else
        echo "0"
    fi
}

# Function to get CPU usage of a specific PID
get_process_cpu() {
    pid=$1
    if [ -n "$pid" ]; then
        ps -o %cpu= -p $pid | awk '{printf "%.2f", $1}'
    else
        echo "0.00"
    fi
}

# Function to get total system memory in MB
get_total_memory() {
    sysctl hw.memsize | awk '{print int($2/1024/1024)}'  # Convert bytes to MB and round to integer
}

echo "Monitoring resource usage for process on port $port. Press Ctrl+C to stop."
echo "Time     | CPU Usage  | Memory Usage"
    
total_memory=$(get_total_memory)

while true; do
    pid=$(get_pid)
    if [ -n "$pid" ]; then
        used_memory=$(get_process_memory $pid)
        cpu_usage=$(get_process_cpu $pid)
        printf "%s | CPU: %s%% | Memory: %d/%d MB\n" "$(date +"%H:%M:%S")" "$cpu_usage" "$used_memory" "$total_memory"
    else
        echo "$(date +"%H:%M:%S") | No process found on port $port"
    fi
    
    sleep 0.5
done