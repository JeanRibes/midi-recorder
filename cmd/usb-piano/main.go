package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	rtmididrv "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

var BPM = float64(120)

func main() {
	inPort := flag.String("input", "LPK25 mk2 MIDI 1", "MIDI input port name")
	outPort := flag.String("output", "Synth input port", "MIDI output port name")

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

	s := smf.New()

	go ui()

	send, err := midi.SendTo(out)
	he(err)
	he(send(midi.ControlChange(0, 64, 127))) //sustain
	/*send(midi.NoteOn(0, 70, 80))
	time.Sleep(100 * time.Millisecond)
	he(send(midi.NoteOff(0, 70)))*/

	main := smf.Track{}
	main.Add(0, smf.MetaTempo(BPM))

	var absmillisec int32 = 0
	TICKS := s.TimeFormat.(smf.MetricTicks)

	shoudStartRecording := false
	isRecording := false

	isSteps := false
	stepIndex := 0
	stepNotes := []uint8{}
	stop, err := midi.ListenTo(in, func(msg midi.Message, absms int32) {
		var ch, key, vel uint8
		switch {
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			on := msg.IsOneOf(midi.NoteOnMsg)

			if isSteps {
				if stepIndex > len(stepNotes)-1 {
					return
				}
				note := stepNotes[stepIndex]
				if on {
					send(midi.NoteOn(ch, note, vel))
				} else {
					send(midi.NoteOn(ch, note, vel))
					stepIndex += 1
				}
				return
			}
			if shoudStartRecording && on {
				// START record
				shoudStartRecording = false
				isRecording = true
				main = smf.Track{}
				main.Add(0, smf.MetaTempo(BPM))
				absmillisec = absms
				/*main.Add(0, msg)
				send(msg)
				return*/
			}
			if isRecording {
				deltams := absms - absmillisec
				absmillisec = absms
				delta := TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
				main.Add(delta, msg)
			}
			send(msg)
		case msg.GetControlChange(&ch, &key, &vel):
			println(ch, key, vel)
			if isSteps {
				stepIndex = 0
			} else {
				if vel == 127 {
					BusFromUItoLoop <- Message{ev: Record}
				}
			}
		}
	})
	he(err)
	quit := make(chan struct{}, 1)

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)
		<-signalCh
		println("interrupt")
		BusFromUItoLoop <- Message{ev: Quit}
	}()
	go func() {
		for {
			msg := <-BusFromUItoLoop
			switch msg.ev {
			case Record:
				println("loop: record SYN")
				if shoudStartRecording && !isRecording {
					shoudStartRecording = false
					BusFromLoopToUI <- Message{ev: Record, boolean: false}
					log.Println("cancel recording")
					continue
				}
				// time.Sleep(time.Second)
				if isRecording { // STOP RECORD
					isRecording = false
					BusFromLoopToUI <- Message{ev: Record, boolean: false}
					log.Println("stop recording")
					main.Close(0)
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					BusFromLoopToUI <- Message{ev: Record, boolean: true}
					log.Println("start recording")
				}
			case PlayPause:
				go func() {
					log.Println("start play")
					var bf bytes.Buffer
					tmpFile := smf.New()
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					he(err)
					player := smf.ReadTracksFrom(&bf)
					he(player.Play(out))
					BusFromLoopToUI <- Message{ev: PlayPause}
					log.Println("end play")
				}()
			case Quit:
				log.Println("quit")
				stop()
				quit <- struct{}{}
			case Quantize:
				go func() {
					log.Printf("quantize at %d BPM\n", msg.number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					main[0].Message = smf.MetaTempo(float64(msg.number))
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					he(quantizer.Quantize(&bf, &bf))
					main = smf.ReadTracksFrom(&bf).SMF().Tracks[0]
					BusFromLoopToUI <- Message{ev: Quantize}
					log.Println("quantize done")
				}()
			case StepMode:
				log.Println("set steps mode to", isSteps)
				isSteps = !isSteps
				stepIndex = 0
				BusFromLoopToUI <- Message{ev: StepMode, boolean: isSteps}
				if isSteps {
					stepNotes = trackToSteps(main)
				}
			case LoadFromFile:
				log.Println("loading file", msg.str)
				file, err := os.Open(msg.str)
				if err != nil {
					log.Println(err)
					BusFromLoopToUI <- Message{ev: Error, str: err.Error()}
					continue
				}
				midiFile := smf.ReadTracksFrom(file).SMF()
				if midiFile == nil || midiFile.NumTracks() < 1 {
					log.Println("empty MIDI file")
					BusFromLoopToUI <- Message{ev: Error, str: "pas un fichier MIDI"}
					continue
				}
				main = midiFile.Tracks[0]
			case SaveToFile:
				fileName := msg.str
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				log.Println("saving to", fileName)
				midiFile := smf.New()
				main.Close(0)
				if err := midiFile.Add(main); err != nil {
					log.Println(err)
					continue
				}
				if err := midiFile.WriteFile(fileName); err != nil {
					log.Println(err.Error())
				}

			}
		}
	}()
	<-quit

	/*
		main.SendTo(TICKS, smf.TempoChanges{},
			func(m midi.Message, timestampms int32) {
				he(send(m))
				deltams := timestampms - absmillisec
				absmillisec = timestampms
				// time.Sleep(TICKS.Duration(BPM, uint32(deltams)) / time.Millisecond)
				// println(deltams)
				println(timestampms)
				_ = deltams
				//t := time.Duration((deltams * int32(time.Millisecond)) / 2)
				//time.Sleep(t)
			})
	*/

}

func he(err error) {
	if err != nil {
		panic(err)
	}
}

func trackToSteps(tr smf.Track) []uint8 {
	steps := []uint8{}
	var ch, key, vel uint8
	for _, ev := range tr {
		if ev.Message.GetNoteOn(&ch, &key, &vel) {
			steps = append(steps, key)
		}
	}
	return steps
}
