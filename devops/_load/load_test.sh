#!/bin/bash

# Flags: --url --ab

URL="http://127.0.0.1:80/"
AB=false

for arg in "$@"; do
    if [ "$arg" == "--url" ]; then
        URL="https://nodo.finance/"
    fi
    if [ "$arg" == "--ab" ]; then
        AB=true
    fi
done

while true; do
    read -p "Requests: " REQUESTS
    if [[ "$REQUESTS" =~ ^[0-9]+$ && "$REQUESTS" -gt 0 ]]; then
        break
    else
        echo "Please enter a positive integer."
    fi
done

if [ "$AB" = "true" ]; then
    while true; do
        read -p "Concurrency: " CONCURRENCY
        if [[ "$CONCURRENCY" =~ ^[0-9]+$ && "$CONCURRENCY" -gt 0 ]]; then
            break
        else
            echo "Please enter a positive integer."
        fi
    done

    echo "Running Apache Benchmark..."
    ab -n $REQUESTS -c $CONCURRENCY $URL
else
    echo "Running Curl..."

    TEMP_FILE=$(mktemp)
    START_TIME=$(date +%s.%N)

    for ((i = 1; i <= $REQUESTS; i++)); do
        echo -ne "Request $i/$REQUESTS\r"

        STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$URL")

        # Save status code to temp file
        echo "$STATUS_CODE" >>"$TEMP_FILE"
    done

    END_TIME=$(date +%s.%N)
    TOTAL_TIME=$(echo "$END_TIME - $START_TIME" | bc)

    echo -e "\n\nStatus Code Summary:"
    echo "URL: $URL"
    echo "Total Time: ${TOTAL_TIME}s"
    echo "-------------------"

    for code in $(sort "$TEMP_FILE" | uniq); do
        count=$(grep -c "^$code$" "$TEMP_FILE")
        percent=$(awk "BEGIN {printf \"%.2f\", ($count * 100 / $REQUESTS)}")

        # Determine status type
        if [ "$code" -ge 200 ] && [ "$code" -lt 300 ]; then
            type="Success"
        elif [ "$code" -ge 300 ] && [ "$code" -lt 400 ]; then
            type="Redirection"
        elif [ "$code" -ge 400 ] && [ "$code" -lt 500 ]; then
            type="Client Error"
        elif [ "$code" -ge 500 ] && [ "$code" -lt 600 ]; then
            type="Server Error"
        else
            type="Unknown"
        fi

        echo "$code ($type): $count ($percent%)"
    done

    rm "$TEMP_FILE"
fi
