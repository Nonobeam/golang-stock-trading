#!/bin/bash

echo "=== Checking Telegram Bot Status ==="
echo ""

# Check if bot process is running
echo "[1] Checking if trading bot process is running..."
ps aux | grep -E "(trading|bot|app)" | grep -v grep
echo ""

# Check Docker containers
echo "[2] Checking Docker containers..."
docker ps -a | grep -E "(trading|bot)"
echo ""

# Test bot API connectivity
echo "[3] Testing Telegram Bot API..."
BOT_TOKEN="8573656924:AAHmqfYufX8cMybax57gCZIV1ivOGhMhclI"
curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getMe" | jq '.'
echo ""

# Check recent updates to bot
echo "[4] Checking recent messages to bot..."
curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getUpdates?limit=5" | jq '.result[-1]'
echo ""

# Check logs (adjust path as needed)
echo "[5] Checking recent logs..."
if [ -f "/var/log/trading-bot.log" ]; then
    tail -n 20 /var/log/trading-bot.log
elif [ -f "./logs/app.log" ]; then
    tail -n 20 ./logs/app.log
else
    echo "Log file not found. Check your log location."
fi
echo ""

echo "=== Diagnostics Complete ==="
