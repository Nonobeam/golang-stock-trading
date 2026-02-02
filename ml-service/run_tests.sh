#!/bin/bash

echo "Running ML Service Test Suite"
echo "=============================="

# Activate virtual environment if exists
if [ -d "venv" ]; then
    if [ -f "venv/Scripts/activate" ]; then
        source venv/Scripts/activate
    elif [ -f "venv/bin/activate" ]; then
        source venv/bin/activate
    fi
fi

# Run all tests with coverage
pytest tests/ \
    --verbose \
    --tb=short \
    --color=yes \
    -x  # Stop on first failure

echo ""
echo "Test run complete!"
