from collections import namedtuple
from copy import deepcopy
import subprocess
import tempfile
from typing import List
from alsa_midi import NoteOffEvent, SequencerClient, READ_PORT,ControlChangeEvent, WRITE_PORT, Event, Address, EventType, PortSubscribedEvent, NoteOnEvent

from alsa_midi.event import NoteEventBase
import os
from time import time_ns, sleep
from midiutil import MIDIFile

client = SequencerClient("step-recorder")
my_input  = client.create_port("input", caps=WRITE_PORT, port_id=0)
clientO = SequencerClient("step-recorder-output")
my_output = clientO.create_port("out", caps=READ_PORT, port_id=0)

if False:
    for available_output in client.list_ports(input=True):
        if available_output.client_name == "AcerPS2":
            port = Address(available_output.client_id, available_output.port_id)
            print(port)
            my_input.connect_from(port)

    for available_input in client.list_ports(output=True):
        if available_input.client_name == "VMPK Input" or available_input.client_name.startswith("FLUID Synth"):
            port = Address(available_input.client_id, available_input.port_id)
            print(port)
            my_output.connect_to(port)

q = client.create_queue()

class SavedEvent(namedtuple("SavedEvent", "event time_ns")):
    pass


def play_record(recorded:List[SavedEvent]):
    for i,_ in enumerate(recorded):
        se = recorded[i]
        clientO.event_output(se.event,port=my_output)
        clientO.drain_output()
        if i < len(recorded)-1:
            sleep(se.time_ns/ 1000000000.0)

def rewrite_offsets(recorded: List[SavedEvent])->List[SavedEvent]:
    i=0
    out = []
    while i < len(recorded)-1:
        cse = recorded[i]
        nse = recorded[i+1]
        out.append(SavedEvent(cse.event, (nse.time_ns - cse.time_ns)))
        i+=1
    return out

def saveMidi(record: List[SavedEvent]):
    m = MIDIFile(deinterleave=False, adjust_origin=False)
    notes = {i:0 for i in range(128)}
    clock = 0
    for se in record:
        ev, toNext = se
        clock += toNext
        note:int = ev.note
        if ev.type == EventType.NOTEON:
            notes[note] = clock
        elif ev.type == EventType.NOTEOFF:
            dur = notes[note] - clock
            m.addNote(time=notes[note],track=0,pitch=note, duration=dur, volume=ev.velocity, channel=0)
    _,name = tempfile.mkstemp(suffix='.mid')
    print(f"writing to {name}")
    with open(name, 'wb+') as f:
        m.writeFile(f)
    print(f"written to {name}")

def note_ping(note, duration=0.1):
    clientO.event_output(NoteOnEvent(note=note,velocity=64), port=my_output)
    clientO.drain_output()
    sleep(duration)
    clientO.event_output(NoteOffEvent(note=note,velocity=64), port=my_output)
    clientO.drain_output()
buffer = [] # store raw notes
record = [] # rewritten offsets
main_record = [] # main audio track

recording = False
while True:
    event: Event = client.event_input()
    if event.source == Address(client.client_id,my_output.port_id):
        continue
    if event.type == EventType.PORT_SUBSCRIBED:
        e: PortSubscribedEvent = event
        print(e)
    elif event.type == EventType.NOTEON or event.type == EventType.NOTEOFF:
        #if event.type == EventType.NOTEON:
        #    _class = NoteOnEvent
        #else:
        #    _class = NoteOffEvent
        #
        ne = event.__class__(event.note,event.channel,event.velocity)
        clientO.event_output(ne, port=my_output)
        clientO.drain_output()
        if recording:
            buffer.append(SavedEvent(ne, time_ns()))
    elif event.type == EventType.CONTROLLER:
        if event.param == 2 and not recording:
                print("start record")
                recording = True
                note_ping(95)
                record_start = time_ns()
        elif recording and (event.param == 2 or event.param == 8):
                clientO.event_output(NoteOnEvent(note=100,velocity=64), port=my_output)
                clientO.drain_output()
                print("stop record")
                recording = False
                buffer.append(SavedEvent(NoteOffEvent(0,velocity=0), time_ns()+500000))
                record.extend(rewrite_offsets(buffer))
                buffer.clear()
                print('stopped')
                clientO.event_output(NoteOffEvent(note=100,velocity=64), port=my_output)
                clientO.drain_output()
        elif not recording and event.param==8: # espace
            record.clear()
            recording = True
            note_ping(95)
            record_start = time_ns()
            print('restarted recording')
        elif event.param==3: # '1'
            recording = False
            print("play record")
            play_record(record)
        elif event.param==6: # '4'
            recording = False
            print("play main record")
            play_record(main_record)
        elif event.param==4: # '2'
            print("reset recoding")
            record.clear()
            recording = False
            note_ping(92)
        elif event.param==5:
            print("appending temp record to main")
            if len(main_record)==0:
                main_record = record.copy()
            else:
                main_record.extend(record)
            note_ping(104)
        elif event.param == 9:
            print("reset main record")
            main_record.clear()
        elif event.param==7:
            print("save main to midi file")
            saveMidi(main_record)
        else:
            ne = ControlChangeEvent(channel=event.channel,param=event.param,value=event.value)
            print(ne)
            clientO.event_output(ne, port=my_output)
            clientO.drain_output()
    else:
        print(event.type)

