#!/bin/bash

# Start ML Prediction Service

# Activate virtual environment if it exists
if [ -d "venv" ]; then
    source venv/bin/activate
fi

# Start the service
python main.py --port 50051
