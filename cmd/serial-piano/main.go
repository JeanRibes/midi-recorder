package main

import (
	"flag"
	"fmt"

	"github.com/albenik/go-serial/v2"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func main() {
	//d, _ := rtmididrv.New()
	//d.OpenVirtualOut("a")
	//drivers.Get().(*rtmididrv.Driver).OpenVirtualIn("aa")

	portName := flag.String("port", "/dev/ttyACM0", "serial port, e.g. /dev/ttyUSB0")
	keymapFile := flag.String("keymap", "keymap.txt", "path of keymap file (format: one 'keycode:note' per line")
	outPort := flag.String("output", "step-recorder", "MIDI output port name")
	//"Synth input port (qsynth:0)"
	debug := flag.Bool("debug", false, "print notes")
	//us := flag.Int("skip", 500, "minimal delay between events")

	flag.Parse()

	keymap := LoadKeymap(*keymapFile)

	port, err := serial.Open(*portName,
		serial.WithBaudrate(115200),
		serial.WithDataBits(8),
		serial.WithParity(serial.NoParity),
		serial.WithStopBits(serial.OneStopBit),
		serial.WithReadTimeout(1000),
		serial.WithWriteTimeout(1000),
		serial.WithHUPCL(false),
	)
	he(err)

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

	{
		buf := make([]byte, 20)
		port.Read(buf)
	}

	state := [256]bool{}
	controller_state := [256]bool{}
	//last_code := 0

	buf := make([]byte, 2)
	//var last_event time.Time
	for {

		n, err := port.Read(buf)
		if err != nil {
			println(err.Error())
			continue
		}
		if n == 0 {
			continue
		}
		if n != 2 {
			println(n)
			//port.Flush()
			continue
		}

		status := int(buf[0])
		code := int(buf[1])

		noteOn := (status >> 7) == 0
		if state[code] == noteOn {
			println("skip double presses")
			continue
		}
		state[code] = noteOn

		/*if noteOn && time.Since(last_event) < time.Microsecond*US {
			println("skip")
			continue
		}
		last_event = time.Now()
		*/

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
