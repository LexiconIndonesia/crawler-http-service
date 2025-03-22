#!/bin/bash

# Script to run the Go application with hot reloading using Air

# Ensure the tmp directory exists
mkdir -p tmp

# Run Air
air -c ./.air.toml
