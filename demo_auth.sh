#!/bin/bash

# GoRedis Authentication Demo Script
# This script demonstrates the login features of GoRedis

echo "=== GoRedis Authentication Demo ==="
echo ""

SERVER_HOST="localhost"
SERVER_PORT="6399"

# Function to send Redis command
send_command() {
    local cmd="$1"
    echo "Sending: $cmd"
    echo -ne "$cmd" | nc $SERVER_HOST $SERVER_PORT
    echo ""
    echo "---"
}

echo "1. Test PING (should work without authentication)"
send_command "*1\r\n\$4\r\nPING\r\n"

echo "2. Test AUTH with admin credentials"
send_command "*3\r\n\$4\r\nAUTH\r\n\$5\r\nadmin\r\n\$8\r\nadmin123\r\n"

echo "3. Test AUTH with wrong password (should fail)"
send_command "*3\r\n\$4\r\nAUTH\r\n\$5\r\nadmin\r\n\$8\r\nwrongpwd\r\n"

echo "4. Test SET command without authentication (should work with default user)"
send_command "*3\r\n\$3\r\nSET\r\n\$4\r\ntest\r\n\$5\r\nvalue\r\n"

echo "5. Test GET command"
send_command "*2\r\n\$3\r\nGET\r\n\$4\r\ntest\r\n"

echo "6. Test ACL LIST (should require authentication)"
send_command "*2\r\n\$3\r\nACL\r\n\$4\r\nLIST\r\n"

echo "Demo completed!"