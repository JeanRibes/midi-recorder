#!/bin/bash
python3 midi-piano.py &
piano=$!

qsynth &
qsynth=$!

python3 step-recorder.py &
recorder=$!

sleep 2

inSynth=$(aconnect -o|grep 'FLUID Synth'|cut -d: -f1|cut -d' ' -f2)
inRecorder=$(aconnect -o|grep 'step-recorder'|cut -d: -f1|cut -d' ' -f2)
outPiano=$(aconnect -i|grep 'AcerPS2'|cut -d: -f1|cut -d' ' -f2)
outRecorder=$(aconnect -i|grep 'step-recorder'|cut -d: -f1|cut -d' ' -f2)

aconnect "$outPiano:0" "$inRecorder:0"
aconnect "$outRecorder:0" "$inSynth:0"

echo "press enter to quit"
read
echo "end"

kill $piano
kill $qsynth
kill $recorder