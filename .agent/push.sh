#!/bin/bash

# Get the current branch name
branch=$(git symbolic-ref --short HEAD)

if [ -z "$branch" ]; then
    echo "Error: Could not determine current branch."
    exit 1
fi

# Push to origin with the current branch name
git push origin "$branch"
