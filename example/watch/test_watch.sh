#!/bin/bash

# Test script for VCFG hot reload demo
echo "🧪 VCFG Hot Reload Test Script"
echo "=============================="

# Reset config to initial state
echo "📝 Resetting config.yaml to initial state..."
cat > config.yaml << EOF
server:
  host: "localhost"
  port: 8080

database:
  host: "localhost"
  port: 5432
  name: "watchdemo"
EOF

echo "✅ Config reset complete"
echo ""
echo "🚀 Starting watch demo in background..."
echo "   (You should see initial configuration displayed)"
echo ""

# Start the watch demo in background
go run main.go &
WATCH_PID=$!

# Wait for the program to start
sleep 3

echo "🔄 Making first configuration change..."
echo "   Changing server port from 8080 to 9090"
sed -i '' 's/port: 8080/port: 9090/' config.yaml
echo "✅ Port changed"

# Wait a bit
sleep 2

echo ""
echo "🔄 Making second configuration change..."
echo "   Changing host from localhost to 127.0.0.1"
sed -i '' 's/localhost/127.0.0.1/g' config.yaml
echo "✅ Host changed"

# Wait a bit
sleep 2

echo ""
echo "🔄 Making third configuration change..."
echo "   Changing database name from watchdemo to newdemo"
sed -i '' 's/watchdemo/newdemo/' config.yaml
echo "✅ Database name changed"

# Wait a bit
sleep 3

echo ""
echo "🛑 Stopping watch demo..."
kill $WATCH_PID
wait $WATCH_PID 2>/dev/null

echo "✅ Test completed!"
echo ""
echo "📋 Final configuration:"
cat config.yaml