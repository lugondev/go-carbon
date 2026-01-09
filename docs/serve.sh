#!/bin/bash

cd "$(dirname "$0")"

echo "Starting Jekyll documentation server..."
echo "Server will be available at: http://127.0.0.1:4000/go-carbon/"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

bundle exec jekyll serve --livereload
