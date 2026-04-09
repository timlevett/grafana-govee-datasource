#!/bin/sh
set -e

echo "Setting up build tools..."

# Install mage if not already installed
if ! command -v mage >/dev/null 2>&1; then
  echo "Installing mage..."
  go install github.com/magefile/mage@latest
fi

# Ensure mage is in PATH
export PATH=$PATH:/root/go/bin

# Configure git safe directory
git config --global --add safe.directory /workspace || true

# Initial builds
echo "Running initial builds..."
cd /workspace
mage -v || echo "Initial mage build failed, continuing..."
npm run build || echo "Initial npm build failed, continuing..."

echo "Watching for changes..."

# Watch Go files and run mage
(
  while true; do
    files=$(find /workspace/pkg /workspace/Magefile.go -type f 2>/dev/null)
    if [ -n "$files" ]; then
      echo "$files" | entr -d -n sh -c 'echo "[Go] Change detected, running mage -v..." && cd /workspace && mage -v' || true
    else
      sleep 2
    fi
  done
) &

# Watch frontend files and run npm build
(
  while true; do
    files=$(find /workspace/src /workspace/package.json /workspace/tsconfig.json -type f 2>/dev/null)
    if [ -n "$files" ]; then
      echo "$files" | entr -d -n sh -c 'echo "[Frontend] Change detected, running npm run build..." && cd /workspace && npm run build' || true
    else
      sleep 2
    fi
  done
) &

# Wait for background processes
wait

