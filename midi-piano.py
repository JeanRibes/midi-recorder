import serial
from alsa_midi import NoteEvent, SequencerClient, READ_PORT, NoteOnEvent, NoteOffEvent,ControlChangeEvent
from time import sleep
from typing import Dict

CHAN = 0
VEL=64

keymap: Dict[int,int] = {i:129 for i in range(0,256)}
with open('midi-keymap.txt', 'r') as f:
    for line in f.readlines():
        k,v = line.split(':')
        key = int(k)
        val = int(v)
        keymap[key]=val

def save_keymap():
    with open('midi-keymap.txt', 'w+') as f:
        for key,value in keymap.items():
            if value !=129:
                f.write(f"{key}:{value}\n")


ps2 = serial.Serial('/dev/ttyACM0', baudrate=115200)

client = SequencerClient("AcerPS2")
port = client.create_port("AcerPS2 output", caps=READ_PORT)
#dest_port = client.list_ports(output=True)[0]
#port.connect_to(dest_port)
#print(ps2.readall())
#ps2.timeout = 1

#client.event_output(NoteEvent(note=65, duration=1000000000))
#client.drain_output()

state = [False] * 256
controller_state = [False] * 256
last_code=0
while True:
    try:
        status, code = ps2.read(2)
    except (TypeError, ValueError):
        continue
    
    noteOn = (status >> 7) == 0
    if state[code] and noteOn and code not in (43,44,45):
        continue
    state[code] = noteOn

    if code in (43,44,45,46) and status == 16: # press + caps lock
        if not noteOn:
            pass
        elif code == 43:
            save_keymap()
        elif code == 44:
            #print(f"set {last_code} to {keymap[last_code] + 1}")
            keymap[last_code] = keymap[last_code] + 1
        elif code == 45:
            #print(f"set {last_code} to {keymap[last_code] - 1}")
            keymap[last_code] = keymap[last_code] - 1
        elif code==46:
            keymap[last_code] = -1
        code = last_code
    elif noteOn:
        last_code = code

    note = keymap[code]
    #print(code, note, status)
    if note < 0 and noteOn:
        print(f"controller {note}")
        if controller_state[code]:
            client.event_output(ControlChangeEvent(channel=CHAN, param=abs(note), value=0))
        else:
            client.event_output(ControlChangeEvent(channel=CHAN, param=abs(note), value=64))
        controller_state[code] = not controller_state[code]
        client.drain_output()
        continue
    elif note < 0 or note > 127:
        print(code, note, status)
        continue
    #print(noteOn * 1, code, note, status, (status >> (4 & 0b00010000))== 1)

    if noteOn:  # 0: MAKE, 1: BREAK
        client.event_output(NoteOnEvent(note=note, velocity=64, channel=CHAN))
    else:
        client.event_output(NoteOffEvent(note=note, velocity=64, channel=CHAN))
    client.drain_output()

# voir typematic rate pour désactive l'autorepeat sur arduino directement
