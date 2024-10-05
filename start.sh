#!/bin/bash
pgrep qsynth || qsynth &
[[ -f piano-assistant ]] || go build ./cmd/piano-assistant
aconnect -d 'LPK25 mk2' 'FLUID Synth (qsynth)'

./piano-assistant