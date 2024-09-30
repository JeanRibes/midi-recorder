package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

func loop(ctx context.Context, cancel func(), in drivers.In, out drivers.Out) {
	LoopDied = false
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level: charmlog.DebugLevel,
		//ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "loop",
	})
	logger.Info("start")

	send, err := midi.SendTo(out)
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{ev: Error, str: "impossible d'ouvrir ce port MIDI en sortie"}
		cancel()
		return
	}
	he(send(midi.ControlChange(0, 64, 127))) //sustain
	/*send(midi.NoteOn(0, 70, 80))
	time.Sleep(100 * time.Millisecond)
	he(send(midi.NoteOff(0, 70)))*/
	if in == nil {
		logger.Error("input port is nil")
		SinkUI <- Message{ev: Error, str: "impossible d'ouvrir ce port MIDI en entrée"}
		cancel()
		return
	}
	logger.Debug("input port", "open", in.IsOpen())

	s := smf.New()
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
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{ev: Error, str: "impossible d'écouter ce port MIDI"}
		cancel()
		return
	}

	LoopDied = true
loopchan:
	for {
		select {
		case <-ctx.Done():
			logger.Debug("context Done")
			break loopchan
		case msg := <-SinkLoop:
			switch msg.ev {
			case Record:
				logger.Debug("record SYN")
				if shoudStartRecording && !isRecording {
					shoudStartRecording = false
					SinkUI <- Message{ev: Record, boolean: false}
					logger.Debug("cancel recording")
					continue
				}
				// time.Sleep(time.Second)
				if isRecording { // STOP RECORD
					isRecording = false
					SinkUI <- Message{ev: Record, boolean: false}
					logger.Debug("stop recording")
					main.Close(0)
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					SinkUI <- Message{ev: Record, boolean: true}
					logger.Debug("start recording")
				}
			case PlayPause:
				go func() {
					logger.Debug("start play")
					var bf bytes.Buffer
					tmpFile := smf.New()
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					player := smf.ReadTracksFrom(&bf)
					he(player.Play(out))
					SinkUI <- Message{ev: PlayPause}
					logger.Debug("end play")
				}()
			case Quantize:
				go func() {
					log.Printf("quantize at %d BPM\n", msg.number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					main[0].Message = smf.MetaTempo(float64(msg.number))
					tmpFile.Add(main)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					he(quantizer.Quantize(&bf, &bf))
					main = smf.ReadTracksFrom(&bf).SMF().Tracks[0]
					SinkUI <- Message{ev: Quantize}
					logger.Debug("quantize done")
				}()
			case StepMode:
				logger.Debug("set steps mode to", isSteps)
				isSteps = !isSteps
				stepIndex = 0
				SinkUI <- Message{ev: StepMode, boolean: isSteps}
				if isSteps {
					stepNotes = trackToSteps(main)
				}
			case LoadFromFile:
				logger.Debug("loading file", msg.str)
				file, err := os.Open(msg.str)
				if err != nil {
					logger.Error(err)
					SinkUI <- Message{ev: Error, str: err.Error()}
					continue
				}
				midiFile := smf.ReadTracksFrom(file).SMF()
				if midiFile == nil || midiFile.NumTracks() < 1 {
					logger.Error("empty MIDI file")
					SinkUI <- Message{ev: Error, str: "pas un fichier MIDI"}
					continue
				}
				main = midiFile.Tracks[0]
			case SaveToFile:
				fileName := msg.str
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				logger.Info("saving to", "filename", fileName)
				midiFile := smf.New()
				main.Close(0)
				if err := midiFile.Add(main); err != nil {
					logger.Error(err)
					continue
				}
				if err := midiFile.WriteFile(fileName); err != nil {
					logger.Error(err)
				}
			case BankStateChange:
				logger.Printf("set bank %d to state %t\n", msg.number, msg.boolean)

			}
		}
	}
	logger.Info("stop")
	if out.IsOpen() {
		stop()
		he(out.Close())
	} else {
		println("already closed")
	}
}
