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

func (r *Recording) Play(send func(msg midi.Message) error) {
	println("playing..")
	for _, note := range *r {
		time.Sleep(note.Time)
		if play_cancel {
			play_cancel = false
			return
		}
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
func (r *Recording) Append(n ...RecordedNote) {
	_r := *r
	_r = append(_r, n...)
	*r = _r
}
func (r *Recording) Reset() {
	*r = Recording{}
}

func Ping(note uint8, send func(msg midi.Message) error) {
	if doPing {
		he(send(midi.NoteOn(config.Channels.Ping, note, 64)))
		time.Sleep(100 * time.Millisecond)
		he(send(midi.NoteOff(config.Channels.Ping, note)))
	}
}

func main() {
	inPort := flag.String("input", "serial-piano", "MIDI input port name")
	outPort := flag.String("output", "", "MIDI output port name")
	flag.BoolVar(&askName, "ask-name", true, "if false, do not ask for a filename on save")
	flag.BoolVar(&doPing, "ping", true, "play 'ping' notes on record/stop/save/append/reset... to confirm user input")
	flag.IntVar(&BPM, "bpm", 120, "MIDI file BPM")
	fileName := flag.String("file", "", "load MIDI file")
	configFile := flag.String("config", "config.yaml", "config file")

	flag.Parse()

	LoadConfig(*configFile)

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

	Ping(60, send)

	recording := false
	var last_time time.Time
	temp_record := Recording{}

	stepIndex := 0

	stepRecording := false
	var step_last_time time.Time
	step_record := SingleRecording{}

	bank_index := 1
	_banks := []*Recording{}
	for i := 0; i < 13; i++ {
		_banks = append(_banks, &Recording{})
	}
	banks := &_banks

	var last_noteon midi.Message

	if *fileName != "" {
		LoadFile(*fileName, &temp_record)
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
	go ui(banks)

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			//fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
			if msg.Is(midi.NoteOnMsg) {
				msg = midi.NoteOn(config.Channels.Output, key, vel)
				last_noteon = msg
			} else {
				msg = midi.NoteOff(config.Channels.Output, key)
			}
			he(send(msg))
			if recording {
				temp_record = append(temp_record, RecordedNote{
					Msg:  msg,
					Time: time.Since(last_time),
				})
				last_time = time.Now()
			}
			return
		case msg.GetControlChange(&ch, &key, &vel):
			//fmt.Printf("control change: %v=%v on chan %v\n", key, vel, ch)
			param := key
			value := vel
			if param == uint8(config.Controllers["sustain_pedal"]) { //sustain
				he(send(msg))
			}
			if param == 0 {
				play_cancel = true
				go Ping(52, send)
				return
			}
			if param == uint8(config.Controllers["record_taps"]) {
				stepRecording = !stepRecording
				if stepRecording {
					step_last_time = time.Now()
					//banks[bank_index].Reset()
					go Ping(95, send)
				} else {
					if len(*(*banks)[bank_index]) > 0 {
						(*(*banks)[bank_index])[0].Time = 0
					}
					go Ping(100, send)
				}
				return
			}
			if (param == config.Controllers["restart_recording"] || param == config.Controllers["start_stop_recording"]) && !recording {
				println("recording...")
				go Ping(95, send)
				recording = true
				last_time = time.Now()
				if param == config.Controllers["restart_recording"] {
					temp_record = temp_record[:0]
				}
				return
			}
			if (param == config.Controllers["restart_recording"] || param == config.Controllers["start_stop_recording"]) && recording {
				go Ping(100, send)
				stopRecording()
				return
			}
			if param == config.Controllers["play_temp"] {
				go temp_record.Play(send)
			}
			if param == config.Controllers["append_temp_to_bank"] {
				//append
				go Ping(104, send)
				//main_record = append(main_record, temp_record...)
				(*banks)[bank_index].Append(temp_record...)
			}
			if param == config.Controllers["play_bank"] {
				//go main_record.Play(send)
				go (*banks)[bank_index].Play(send)
			}
			if param == config.Controllers["del_temp"] {
				println("reset")
				go Ping(92, send)
				//temp_record = temp_record[:0]
				temp_record.Reset()
				stopRecording()
				//reset
			}
			if param == config.Controllers["del_bank"] {
				println("del")
				go Ping(45, send)
				//main_record = main_record[:0]
				(*banks)[bank_index].Reset()
			}
			if param == config.Controllers["save_bank"] {
				println("saving..")
				//Save(&main_record)
				Save((*banks)[bank_index], "")
				println("done")
			}

			if param == config.Controllers["reset_step_index"] {
				println("step reset")
				stepIndex = 0
				go Ping(64, send)
			}

			if param == config.Controllers["step_next"] { // step playing
				println("step", stepIndex)
				if len(step_record) == 0 {
					return
				}
				if stepIndex < 0 {
					stepIndex = 0
				}

				if stepIndex < len(step_record) {
					ev := step_record[stepIndex]
					var msg midi.Message
					if value > 0 {
						msg = midi.NoteOn(config.Channels.Output, ev.Note, value) // value = 64
					} else {
						msg = midi.NoteOff(config.Channels.Output, ev.Note)
						stepIndex += 1
					}
					send(msg)
					if stepRecording {
						(*banks)[bank_index].Append(RecordedNote{
							Msg:  msg,
							Time: time.Since(step_last_time),
						})
						step_last_time = time.Now()
					}
				} else {
					if value > 0 {
						go Ping(94, send)
					}
				}
			}
			if param == config.Controllers["step_previous"] {
				if len(step_record) == 0 {
					return
				}
				if stepIndex >= len(step_record) {
					stepIndex = len(step_record) - 1
				}
				if stepIndex >= 0 {
					ev := step_record[stepIndex]
					if value > 0 {
						send(midi.NoteOn(config.Channels.Output, ev.Note, value)) // value = 64
					} else {
						send(midi.NoteOff(config.Channels.Output, ev.Note))
						stepIndex -= 1
					}
				} else {
					if value > 0 {
						go Ping(92, send)
					}
				}
			}
			if param == config.Controllers["load_bank_to_step"] { // put bank back into temp
				temp_record = append(temp_record, *(*banks)[bank_index]...)
				step_record = temp_record.RemoveChords1()
				println("bank back into temp")
				go Ping(104, send)
			}
			if param == config.Controllers["bank_select"] && value > 0 { // bank select: F1 -> F12
				bank_index = int(value)
			}
			if param == config.Controllers["filter_chords_jump"] { //inser
				step_record = temp_record.RemoveChords1()
			}
			if param == config.Controllers["filter_chords_ignore"] { //suppr
				step_record = temp_record.RemoveChords2()
			}
			if param == config.Controllers["delete_step"] {
				stepIndex -= 1
				if stepIndex >= 0 && stepIndex < len(step_record) {
					step_record = append(step_record[:stepIndex], step_record[stepIndex+1:]...)
				}
			}
			if param == config.Controllers["incremental_add"] { // add notes one by one
				last_noteon.Is(midi.NoteOnMsg)
				var note uint8
				var _d uint8
				if last_noteon.GetNoteOn(&_d, &note, &_d) {
					step_record = append(step_record, NoteOnOff{
						Note:     note,
						Duration: time.Millisecond * 300,
					})
					temp_record = append(temp_record, RecordedNote{
						Msg:  midi.NoteOn(config.Channels.Output, note, 64),
						Time: time.Millisecond * 200,
					})
					temp_record = append(temp_record, RecordedNote{
						Msg:  midi.NoteOff(config.Channels.Output, note),
						Time: time.Millisecond * 300,
					})
				}
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

func Save(recording *Recording, name string) {
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

func LoadFile(fileName string, recording *Recording) {
	mid, err := smf.ReadFile(fileName)
	he(err)
	println(mid.NumTracks())
	track := mid.Tracks[0]
	println("track 0 len:", len(track))
	recording.Reset()
	_r := *recording
	//temp_record = temp_record[:0]
	bpm := float64(BPM)
	clock := mid.TimeFormat.(smf.MetricTicks)
	for _, ev := range track {
		if ev.Message.GetMetaTempo(&bpm) {
			continue
		}
		if ev.Message.IsOneOf(midi.NoteOffMsg, midi.NoteOnMsg) {
			_r = append(_r, RecordedNote{
				Msg:  ev.Message.Bytes(),
				Time: clock.Duration(bpm, ev.Delta),
			})

		} else {
			println(ev.Message.String())
		}
	}
	if len(_r) > 0 {
		_r[0].Time = 0
	}
	println("main:", len(_r))
	*recording = _r
}
