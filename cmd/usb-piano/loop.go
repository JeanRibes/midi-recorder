package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

func loop(ctx context.Context, cancel func(), in drivers.In, out drivers.Out) {

	s := smf.New()

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
					SinkLoop <- Message{ev: Record}
				}
			}
		}
	})
	he(err)

loopchan:
	for {
		select {
		case <-ctx.Done():
			log.Println("loop: context Done")
			break loopchan
		case msg := <-SinkLoop:
			switch msg.ev {
			case Record:
				println("loop: record SYN")
				if shoudStartRecording && !isRecording {
					shoudStartRecording = false
					SinkUI <- Message{ev: Record, boolean: false}
					log.Println("cancel recording")
					continue
				}
				// time.Sleep(time.Second)
				if isRecording { // STOP RECORD
					isRecording = false
					SinkUI <- Message{ev: Record, boolean: false}
					log.Println("stop recording")
					main.Close(0)
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					SinkUI <- Message{ev: Record, boolean: true}
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
					SinkUI <- Message{ev: PlayPause}
					log.Println("end play")
				}()
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
					SinkUI <- Message{ev: Quantize}
					log.Println("quantize done")
				}()
			case StepMode:
				log.Println("set steps mode to", isSteps)
				isSteps = !isSteps
				stepIndex = 0
				SinkUI <- Message{ev: StepMode, boolean: isSteps}
				if isSteps {
					stepNotes = trackToSteps(main)
				}
			case LoadFromFile:
				log.Println("loading file", msg.str)
				file, err := os.Open(msg.str)
				if err != nil {
					log.Println(err)
					SinkUI <- Message{ev: Error, str: err.Error()}
					continue
				}
				midiFile := smf.ReadTracksFrom(file).SMF()
				if midiFile == nil || midiFile.NumTracks() < 1 {
					log.Println("empty MIDI file")
					SinkUI <- Message{ev: Error, str: "pas un fichier MIDI"}
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
			case BankStateChange:
				log.Printf("set bank %d to state %t\n", msg.number, msg.boolean)

			}
		}
	}
	log.Println("loop: quit")
	stop()
}
