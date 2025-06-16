#!/bin/bash
##### DISCLAIMER : ChatGPT helped write this code, but we just used it to automate the process of setting up our test suite (it just opens the windows and calls the chat program to save time)

NUM_WINDOWS=$1
TESTFILE=$2

EXT_DISPLAY_X=0
EXT_DISPLAY_Y=0
EXT_DISPLAY_WIDTH=2040
EXT_DISPLAY_HEIGHT=1280

TOP_MARGIN=22
BOTTOM_MARGIN=0
V_GAP=40

SCREEN_WIDTH=$EXT_DISPLAY_WIDTH
SCREEN_HEIGHT=$EXT_DISPLAY_HEIGHT

AVAILABLE_HEIGHT=$(( SCREEN_HEIGHT - TOP_MARGIN - BOTTOM_MARGIN ))

if [ "$NUM_WINDOWS" -le 4 ]; then
    COLS=2
    ROWS=2
elif [ "$NUM_WINDOWS" -le 8 ]; then
    COLS=2
    ROWS=4
elif [ "$NUM_WINDOWS" -le 16 ]; then
    COLS=4
    ROWS=4
else
    echo "Error: Maximum 16 windows supported."
    exit 1
fi

TOTAL_GAP=$(( (ROWS - 1) * V_GAP ))
WINDOW_WIDTH=$(( SCREEN_WIDTH / COLS ))
WINDOW_HEIGHT=$(( (AVAILABLE_HEIGHT - TOTAL_GAP) / ROWS ))

echo "Starting nodesetd (go run lib/nodeset/cmd/nodesetd/nodesetd.go)..."

NODESETD_LOG=$(mktemp /tmp/nodesetd.log.XXXXXX)

go run lib/nodeset/cmd/nodesetd/nodesetd.go > "$NODESETD_LOG" &
NODESETD_PID=$!

echo "Waiting for nodesetd to output IP:port..."
while true; do
    if [ -s "$NODESETD_LOG" ]; then
        # Read the first line from the log file.
        read -r IPADDR < "$NODESETD_LOG"
        if [ -n "$IPADDR" ]; then
            echo "nodesetd is running at: $IPADDR"
            break
        fi
    fi
    sleep 0.2
done

# Global array to store spawned iTerm window IDs
window_ids=()

# cleanup() is called when the script receives SIGINT or SIGTERM.
cleanup() {
    echo "Cleaning up spawned terminals..."
    for wid in "${window_ids[@]}"; do
        osascript <<EOF
tell application "iTerm"
    try
        set theWindow to first window whose id is $wid
        close theWindow
    end try
end tell
EOF
    done
    # Also, if nodesetd is still running, terminate it.
    if kill -0 "$NODESETD_PID" 2>/dev/null; then
        kill "$NODESETD_PID"
    fi
    exit 0
}

# Trap SIGINT (Ctrl-C) and SIGTERM so that cleanup() runs when the script is killed.
trap cleanup SIGINT SIGTERM

open_terminal() {
    local index=$1
    local row=$(( index / COLS ))
    local col=$(( index % COLS ))
    # Compute horizontal position relative to the external display.
    local posX=$(( 0 - EXT_DISPLAY_X + col * WINDOW_WIDTH ))
    # Compute vertical position (starting at TOP_MARGIN and adding row offset plus gap).
    local posY=$(( 0 - EXT_DISPLAY_Y + TOP_MARGIN + row * WINDOW_HEIGHT + V_GAP ))
    local right=$(( posX + WINDOW_WIDTH ))
    local bottom=$(( posY + WINDOW_HEIGHT ))

    # Create a new iTerm window and capture its ID.
    local newWindowID
    newWindowID=$(osascript <<EOF
tell application "iTerm"
    set newWindow to (create window with default profile)
    set newWindowID to id of newWindow
    tell current session of newWindow
        write text "go run cmd/testChat/testChat.go $IPADDR $TESTFILE"
    end tell
    set bounds of newWindow to {$posX, $posY, $right, $bottom}
    return newWindowID
end tell
EOF
)
    # Trim any extraneous whitespace/newlines.
    newWindowID=$(echo "$newWindowID" | tr -d '[:space:]')
    window_ids+=("$newWindowID")
}

for (( i=0; i<NUM_WINDOWS; i++ )); do
    open_terminal "$i" &
done

wait
