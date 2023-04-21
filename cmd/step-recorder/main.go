package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
	"gitlab.com/gomidi/midi/v2/smf"
)

var BPM int
var doPing bool
var askName bool

type RecordedNote struct {
	Msg  midi.Message
	Time time.Duration
}

type Recording []RecordedNote

func (r Recording) Play(send func(msg midi.Message) error) {
	println("playing..")
	for _, note := range r {
		time.Sleep(note.Time)
		send(note.Msg)
	}
	println("finished")
}

func Ping(note uint8, send func(msg midi.Message) error) {
	if doPing {
		he(send(midi.NoteOn(0, note, 64)))
		time.Sleep(100 * time.Millisecond)
		he(send(midi.NoteOff(0, note)))
	}
}

func main() {
	inPort := flag.String("input", "serial-piano", "MIDI input port name")
	outPort := flag.String("output", "", "MIDI output port name")
	flag.BoolVar(&askName, "ask-name", true, "if false, do not ask for a filename on save")
	flag.BoolVar(&doPing, "ping", true, "play 'ping' notes on record/stop/save/append/reset... to confirm user input")
	flag.IntVar(&BPM, "bpm", 120, "MIDI file BPM")

	flag.Parse()

	defer midi.CloseDriver()
	drv := drivers.Get().(*rtmididrv.Driver)
	in, err := midi.FindInPort(*inPort)
	if err != nil {
		fmt.Println("can't find input, opening one")
		in, err = drv.OpenVirtualIn("step-recorder")
		he(err)
	}
	println("input:", in.String())

	out, err := midi.FindOutPort(*outPort)
	if err != nil {
		fmt.Println("can't find output")
		out, err = drv.OpenVirtualOut("step-recorder")
		he(err)
	}
	println("output:", out.String())
	send, err := midi.SendTo(out)
	he(err)

	temp_record := Recording{}
	main_record := Recording{}

	var last_time time.Time
	recording := false

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			//fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
			he(send(msg))
			if recording {
				temp_record = append(temp_record, RecordedNote{
					Msg:  msg,
					Time: time.Since(last_time),
				})
				last_time = time.Now()
			}
		case msg.GetNoteEnd(&ch, &key):
			//fmt.Printf("ending note %s on channel %v\n", midi.Note(key), ch)
			he(send(msg))
		case msg.GetControlChange(&ch, &key, &vel):
			//fmt.Printf("control change: %v=%v on chan %v\n", key, vel, ch)
			param := key
			//value := vel
			if param == 64 { //sustain
				he(send(msg))
			}
			if param == 2 && !recording {
				println("recording...")
				go Ping(95, send)
				recording = true
				last_time = time.Now()
				return
			}
			if param == 8 && !recording {
				println("recording...")
				go Ping(95, send)
				recording = true
				last_time = time.Now()
				temp_record = temp_record[:0]
				return
			}
			if (param == 8 || param == 2) && recording {
				println("stopped recording")
				go Ping(100, send)
				recording = false
				if len(temp_record) > 0 {
					temp_record[0].Time = 0
				}
				return
			}
			if param == 3 {
				go temp_record.Play(send)
			}
			if param == 5 {
				//append
				go Ping(104, send)
				main_record = append(main_record, temp_record...)
			}
			if param == 6 {
				go main_record.Play(send)
			}
			if param == 4 {
				println("reset")
				go Ping(92, send)
				temp_record = temp_record[:0]
				//reset
			}
			if param == 9 {
				println("del")
				go Ping(45, send)
				main_record = main_record[:0]
			}
			if param == 7 {
				println("saving..")
				Save(&main_record)
				println("done")
			}

		default:
			// ignore
		}
	}, midi.UseSysEx())
	he(err)
	println("waiting for keyboard input")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	<-signalCh
	println("stop")
	stop()
}

func Save(recording *Recording) {
	file := smf.New()
	bpm := float64(BPM) // can be anything ?
	main := smf.Track{}
	main.Add(0, smf.MetaTrackSequenceName("main"))
	//main.Add(0, smf.MetaMeter(3, 4))
	main.Add(0, smf.MetaTempo(bpm))
	clock := file.TimeFormat.(smf.MetricTicks)
	for _, note := range *recording {
		tick := clock.Ticks(bpm, note.Time)
		main.Add(tick, note.Msg)
	}
	main.Close(0)
	file.Add(main)

	name := AskName()
	if name == "" {
		name = "recording-" + time.Now().Format("2006-01-02-15:04:05") + ".mid"
	}

	he(file.WriteFile(name))
	println("saved to", name)
}

func AskName() string {
	if !askName {
		return ""
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choose a name for the recording: ")
	name, err := reader.ReadString('\n')
	if err != nil || len(name) < 2 {
		println(err)
		return ""
	} else {
		name = name[0:len(name)-1] + ".mid"
		return name
	}
}

func he(err error) {
	if err != nil {
		println("Error: ", err)
	}
}
