#!/bin/sh
set -e

echo "running migrations..."
./migrate up

echo "starting server..."
exec ./server
