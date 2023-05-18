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

const NUM_BANKS = 12

var play_cancel = false

type RecordedNote struct {
	Msg  midi.Message
	Time time.Duration
}

type Recording []RecordedNote

func (r Recording) Play(send func(msg midi.Message) error) {
	println("playing..")
	for _, note := range r {
		if play_cancel {
			play_cancel = false
			return
		}
		time.Sleep(note.Time)
		send(note.Msg)
		/*
			var ch, key, vel uint8
			if note.Msg.GetNoteOn(&ch, &key, &vel) {
				println(ch, key, vel, note.Time, "on")
			}
			if note.Msg.GetNoteOff(&ch, &key, &vel) {
				println(ch, key, vel, note.Time, "off")
			}
		*/
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
	fileName := flag.String("file", "", "load MIDI file")

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

	recording := false
	var last_time time.Time
	temp_record := Recording{}

	stepIndex := 0

	stepRecording := false
	var step_last_time time.Time
	step_record := SingleRecording{}

	bank_index := 1
	banks := [NUM_BANKS + 1]Recording{}

	if *fileName != "" {
		mid, err := smf.ReadFile(*fileName)
		he(err)
		println(mid.NumTracks())
		track := mid.Tracks[0]
		println("track 0 len:", len(track))
		temp_record = temp_record[:0]
		//temp_record = temp_record[:0]
		bpm := float64(BPM)
		clock := mid.TimeFormat.(smf.MetricTicks)
		for _, ev := range track {
			if ev.Message.GetMetaTempo(&bpm) {
				continue
			}
			if ev.Message.IsOneOf(midi.NoteOffMsg, midi.NoteOnMsg) {
				temp_record = append(temp_record, RecordedNote{
					Msg:  ev.Message.Bytes(),
					Time: clock.Duration(bpm, ev.Delta),
				})

			} else {
				println(ev.Message.String())
			}
		}
		if len(temp_record) > 0 {
			temp_record[0].Time = 0
		}
		println("main:", len(temp_record))
	}

	stopRecording := func() {
		println("stopped recording")
		recording = false
		if len(temp_record) > 0 {
			temp_record[0].Time = 0
		}
		step_record = temp_record.RemoveChords2()
		println("step", len(step_record))
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			//fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
			he(send(msg))
			if recording {
				temp_record = append(temp_record, RecordedNote{
					Msg:  msg,
					Time: time.Since(last_time),
				})
				last_time = time.Now()
			}
		case msg.GetControlChange(&ch, &key, &vel):
			//fmt.Printf("control change: %v=%v on chan %v\n", key, vel, ch)
			param := key
			value := vel
			if param == 64 { //sustain
				he(send(msg))
			}
			if param == 0 {
				play_cancel = true
				go Ping(52, send)
				return
			}
			if param == 19 {
				stepRecording = !stepRecording
				if stepRecording {
					step_last_time = time.Now()
					//banks[bank_index].Reset()
					go Ping(95, send)
				} else {
					if len(banks[bank_index]) > 0 {
						banks[bank_index][0].Time = 0
					}
					go Ping(100, send)
				}
				return
			}
			if (param == 8 || param == 2) && !recording {
				println("recording...")
				go Ping(95, send)
				recording = true
				last_time = time.Now()
				temp_record = temp_record[:0]
				return
			}
			if (param == 8 || param == 2) && recording {
				go Ping(100, send)
				stopRecording()
				return
			}
			if param == 3 {
				go temp_record.Play(send)
			}
			if param == 5 {
				//append
				go Ping(104, send)
				//main_record = append(main_record, temp_record...)
				banks[bank_index] = append(banks[bank_index], temp_record...)
			}
			if param == 6 {
				//go main_record.Play(send)
				go banks[bank_index].Play(send)
			}
			if param == 4 {
				println("reset")
				go Ping(92, send)
				//temp_record = temp_record[:0]
				temp_record.Reset()
				stopRecording()
				//reset
			}
			if param == 9 {
				println("del")
				go Ping(45, send)
				//main_record = main_record[:0]
				banks[bank_index] = banks[bank_index][:0]
			}
			if param == 7 {
				println("saving..")
				//Save(&main_record)
				Save(&banks[bank_index])
				println("done")
			}

			if param == 10 {
				println("step reset")
				stepIndex = 0
				go Ping(64, send)
			}

			if param == 11 { // step playing
				println("step", stepIndex)
				if len(step_record) == 0 {
					return
				}
				if stepIndex < 0 {
					stepIndex = 0
				}

				if stepIndex < len(step_record) && stepIndex >= 0 {
					ev := step_record[stepIndex]
					var msg midi.Message
					if value > 0 {
						msg = midi.NoteOn(0, ev.Note, value) // value = 64
					} else {
						msg = midi.NoteOff(0, ev.Note)
						stepIndex += 1
					}
					send(msg)
					if stepRecording {
						banks[bank_index] = append(banks[bank_index], RecordedNote{
							Msg:  msg,
							Time: time.Since(step_last_time),
						})
						step_last_time = time.Now()
					}
				}
			}
			if param == 12 {
				if len(step_record) == 0 {
					return
				}
				if stepIndex >= len(step_record) {
					stepIndex = len(step_record) - 1
				}
				if stepIndex > 0 {
					ev := step_record[stepIndex]
					if value > 0 {
						send(midi.NoteOn(0, ev.Note, value)) // value = 64
					} else {
						send(midi.NoteOff(0, ev.Note))
						stepIndex -= 1
					}
				}
			}
			if param == 13 { // put bank back into temp
				temp_record = append(temp_record, banks[bank_index]...)
				step_record = temp_record.RemoveChords1()
				println("bank back into temp")
				go Ping(104, send)
			}
			if param == 16 && value > 0 { // bank select: F1 -> F12
				bank_index = int(value)
			}
			if param == 17 { //inser
				step_record = temp_record.RemoveChords1()
			}
			if param == 18 { //suppr
				step_record = temp_record.RemoveChords2()
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

func (r *Recording) Reset() {
	_r := *r
	*r = _r[:0]
}
