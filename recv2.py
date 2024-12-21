from mido import Message, Backend, open_output, open_input, MidiTrack, MidiFile
from mido import second2tick, tick2second
from mido.ports import BaseInput, BaseOutput
from time import sleep, time_ns, time
import os

rtmidi = Backend("mido.backends.rtmidi")

PORT_IN_NAME = "RecorderInput"
PORT_OUT_NAME = "RecorderOutput"
TICKS_PER_BEAT = 480
TEMPO = 500000

out: BaseOutput = open_output("out", client_name=PORT_OUT_NAME)  #'FLUID Synth'
os.system(f"aconnect '{PORT_OUT_NAME}:0' 'FLUID Synth:0'")
inp: BaseInput = open_input("in", client_name=PORT_IN_NAME)  # virtual=True #AcerPS2
os.system(f"aconnect 'AcerPS2:0' '{PORT_IN_NAME}:0'")
out.panic()  # stop all notes

temp_record = MidiTrack()
main_record = MidiTrack()
MidiFile()
recording = False
record_offset = 0


def start_recording():
    global recording, record_offset
    print("start recording")
    recording = True
    record_offset = time()


def reset_recording():
    temp_record.clear()


def append_recording():
    pass


def stop_recording():
    print("stop recording")
    global recording
    recording = False


def play_recording(record: MidiTrack):
    print("playing")
    start_time = time()
    input_time = 0.0

    for msg in record:
        print(msg)
        input_time += msg.time

        playback_time = time() - start_time
        duration_to_next_event = tick2second(input_time - playback_time, ticks_per_beat=TICKS_PER_BEAT, tempo=TEMPO)

        if duration_to_next_event > 0.0:
            sleep(duration_to_next_event)

        if not msg.is_meta:
            out.send(msg.copy())
    print("stop")
def play_recording2(record: MidiTrack):
    for msg in record:
        out.send(msg)
        sleep(tick2second(msg.time, ticks_per_beat=TICKS_PER_BEAT, tempo=TEMPO)/2)
    #return
    mf = MidiFile(tracks=record.copy())
    for msg in mf.play():
        out.send(msg)


while not inp.closed:
    msg: Message = inp.receive(block=True)

    if msg.type == "note_on" or msg.type == "note_off":
        out.send(msg)
        if recording:
            cm = msg.copy()
            cm.time = second2tick(
                time() - record_offset, ticks_per_beat=TICKS_PER_BEAT, tempo=TEMPO
            )
            temp_record.append(cm)
    elif msg.type == "control_change" and msg.control == 64:  # sustain pedal
        out.send(msg)
    elif msg.type == "control_change":
        if msg.control == 2:
            if recording:
                stop_recording()
            else:
                start_recording()
        elif msg.control == 4:
            reset_recording()
        elif msg.control == 3:
            play_recording(temp_record)
        elif msg.control == 6:
            play_recording(main_record)

    else:
        print(msg)
