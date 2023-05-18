#!/usr/bin/env bash
cd "$(dirname "$0")"
./serial-piano -keymap keymap.txt -port /dev/ttyACM0 &
ppid=$!
qsynth &
sleep 1
./step-recorder -input serial-piano -output 'Synth input port (qsynth:0)' &
rpid=$!
qpid=$!
function quit {
	kill $ppid
	kill $rpid
	kill $qpid
	kill -9 $ppid || true
	kill -9 $rpid || true
	kill -9 $qpid || true
	exit 0
	echo "quit !"
}
trap quit INT
read
quit
