#!/usr/bin/env bash

./serial-piano -keymap keymap.txt -port /dev/ttyACM0 &
ppid=$!
sleep 1
./step-recorder -input serial-piano -output 'Synth input port (qsynth:0)' &
rpid=$!
function quit {
	kill $ppid
	kill $rpid
	kill -9 $ppid || true
	kill -9 $rpid || true
	exit 0
	echo "quit !"
}
trap quit INT
read
quit
