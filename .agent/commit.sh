#!/bin/bash

# Check if a commit message is provided
if [ -z "$1" ]; then
  echo "Error: Commit message is required."
  echo "Usage: $0 \"Your commit message\""
  exit 1
fi

# Add all changes
git add .

# Commit with the provided message
git commit -m "$1"
