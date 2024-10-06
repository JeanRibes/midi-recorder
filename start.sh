#!/bin/bash
pgrep qsynth || qsynth &
while [[ $(aconnect -l|grep qsynth|wc -l) -lt 2 ]]; do
    sleep 1
    echo -n "."
done
echo "ok"
[[ -f piano-assistant ]] || go build ./cmd/piano-assistant
aconnect -d 'LPK25 mk2' 'FLUID Synth (qsynth)'
echo "disconnected default"
./piano-assistant