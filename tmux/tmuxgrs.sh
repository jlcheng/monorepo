#!/bin/bash

# Launches grs and its log file in tmux


SESSION="grs"
LOG_FILE="$HOME/tmp/grs.log"

tmux has-session -t $SESSION
ok="$?"
if [ "$ok" != "0" ]; then
    tmux -2 new-session -d -s $SESSION

    tmux select-window -t $SESSION:1 -n 'Grs'
    tmux split-window -v
    tmux select-pane -t 0
    tmux send-keys "grs -v --log-file $LOG_FILE" C-m

    tmux select-pane -t 1
    tmux send-keys "tail -f $LOG_FILE" C-m
    tmux resize-pane -D 20
    tmux select-pane -t 0

    tmux new-window -n "bash"

    tmux select-window -t $SESSION:1
    tmux -2 attach-session -d -t $SESSION
else
    tmux -2 attach-session -d -t $SESSION
fi
