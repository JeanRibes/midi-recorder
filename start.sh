#!/usr/bin/env bash
cd "$(dirname "$0")"
./cmd/serial-piano/serial-piano -keymap config.yaml -port /dev/ttyACM0 &
ppid=$!
qsynth &
qpid=$!
sleep 2
#./step-recorder -input serial-piano -output 'Synth input port (qsynth:0)' & ; rpid=$!
./cmd/step-recorder/step-recorder -input serial-piano -output 'Synth input port (qsynth:0)'

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
