package main

import (
	"flag"
	"fmt"
	"log"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
	"go.bug.st/serial"
)

func main() {
	//d, _ := rtmididrv.New()
	//d.OpenVirtualOut("a")
	//drivers.Get().(*rtmididrv.Driver).OpenVirtualIn("aa")

	portName := flag.String("port", "/dev/ttyACM0", "serial port, e.g. /dev/ttyUSB0")
	keymapFile := flag.String("keymap", "keymap.txt", "path of keymap file (format: one 'keycode:note' per line")
	outPort := flag.String("output", "", "MIDI output port name")
	debug := flag.Bool("debug", false, "print notes")
	//"Synth input port (qsynth:0)"

	flag.Parse()

	keymap := LoadKeymap(*keymapFile)

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}

	if *portName == "" {
		portName = &ports[0]
	}

	mode := &serial.Mode{
		BaudRate: 115200,
	}
	port, err := serial.Open(*portName, mode)
	if err != nil {
		log.Fatal(err)
	}

	/*if *outPort == "" {
		np, err := rtmidi.NewMIDIIn(rtmidi.APILinuxALSA, "testout", 1)
		he(err)
		np.OpenPort(0, "dummy-port")
		he(err)
		*outPort = "dummy-port"
	}*/
	println(midi.GetOutPorts().String())

	//out, err := midi.OutPort(1)
	out, err := midi.FindOutPort(*outPort)
	if err != nil {
		fmt.Println("can't find output", *outPort, "opening a new one")
		//		out, err = midi.OutPort(0)
		out, err = drivers.Get().(*rtmididrv.Driver).OpenVirtualOut("serial-piano")

		he(err)
	}

	println("output:", out.String())
	send, err := midi.SendTo(out)
	he(err)

	state := [256]bool{}
	controller_state := [256]bool{}
	//last_code := 0

	port.ResetInputBuffer()
	buf := make([]byte, 2)
	for {
		_, err := port.Read(buf)
		if err != nil {
			println(err)
		}
		status := int(buf[0])
		code := int(buf[1])

		noteOn := (status >> 7) == 0
		if state[code] && noteOn {
			continue
		}
		state[code] = noteOn

		note, ok := keymap[code]
		if note < 0 && noteOn && ok {
			if *debug {
				fmt.Printf("controller %d\n", -1*note)
			}
			if controller_state[code] {
				send(midi.ControlChange(0, uint8(-1*note), 0))
			} else {
				send(midi.ControlChange(0, uint8(-1*note), 64))
			}
			controller_state[code] = !controller_state[code]
			continue
		}
		if note >= 0 && note <= 127 && ok {
			if *debug {
				fmt.Printf("note %d: %t\n", note, noteOn)
			}
			if noteOn {
				send(midi.NoteOn(0, uint8(note), 64))
			} else {
				send(midi.NoteOff(0, uint8(note)))
			}
		}

		if !ok && *debug {
			fmt.Printf("unassigned: %d\n", code)
		}
	}
}
