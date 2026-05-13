#!/bin/bash
# Simple wrapper script for installing Paca plugins
# This script provides a shortcut interface to the main install-local-plugin.sh script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAIN_SCRIPT="$SCRIPT_DIR/install-local-plugin.sh"

# Check if main script exists
if [[ ! -f "$MAIN_SCRIPT" ]]; then
    echo "Error: Main script not found at $MAIN_SCRIPT"
    exit 1
fi

# Pass all arguments to the main script
exec "$MAIN_SCRIPT" "$@"
