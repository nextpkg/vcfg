#!/bin/bash
# Test script for configuration reload functionality
# This script automatically modifies config and restores it for repeatable testing

set -e  # Exit on any error

echo "=== Configuration Reload Test Script ==="
echo "This script will test the configuration reload functionality"
echo

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "Error: config.yaml not found in current directory"
    exit 1
fi

# Backup original configuration
echo "1. Backing up original configuration..."
cp config.yaml config.yaml.backup
echo "   ✓ Backup created: config.yaml.backup"

# Start the application in background
echo "2. Starting application..."
go run . &
APP_PID=$!
echo "   ✓ Application started with PID: $APP_PID"

# Wait for application to fully start
echo "3. Waiting for application to initialize..."
sleep 2

# Modify configuration to trigger reload
echo "4. Modifying configuration to trigger reload..."
# Change kafka_producer bootstrap_servers from localhost:9095 to localhost:9096
sed -i 's/localhost:9095/localhost:9096/g' config.yaml
echo "   ✓ Changed kafka_producer bootstrap_servers: localhost:9095 -> localhost:9096"

# Wait to observe the reload
echo "5. Waiting to observe configuration reload..."
sleep 2

# Modify configuration again to test multiple reloads
echo "6. Testing second configuration change..."
# Change kafka_consumer bootstrap_servers from localhost:9093 to localhost:9097
sed -i 's/localhost:9093/localhost:9097/g' config.yaml
echo "   ✓ Changed kafka_consumer bootstrap_servers: localhost:9093 -> localhost:9097"

# Wait to observe the second reload
echo "7. Waiting to observe second configuration reload..."
sleep 2

# Stop the application
echo "8. Stopping application..."
kill $APP_PID 2>/dev/null || true
wait $APP_PID 2>/dev/null || true
echo "   ✓ Application stopped"

# Restore original configuration
echo "9. Restoring original configuration..."
mv config.yaml.backup config.yaml
echo "   ✓ Original configuration restored"

echo
echo "=== Test completed successfully! ==="
echo "You can run this script multiple times to test configuration reload."
echo "Usage: ./test_config_reload.sh"