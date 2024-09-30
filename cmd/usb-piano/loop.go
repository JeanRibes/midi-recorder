package main

import (
	"bytes"
	"context"
	"errors"
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

var recordTrack smf.Track

func init() {
	recordTrack.Add(0, smf.MetaTempo(BPM))
}

func loop(ctx context.Context, cancel func(), in drivers.In, out drivers.Out) {
	LoopDied = false
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level: charmlog.DebugLevel,
		//ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "loop",
	})
	logger.Info("start")
	logger.Info("connecting to", "input", in.String())
	logger.Info("connecting to", "output", out.String())

	send, err := midi.SendTo(out)
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{ev: Error, str: "impossible d'ouvrir ce port MIDI en sortie"}
		cancel()
		return
	}
	he(send(midi.Reset()))
	he(send(midi.ControlChange(0, 64, 127))) //sustain
	if in == nil {
		logger.Error("input port is nil")
		SinkUI <- Message{ev: Error, str: "impossible d'ouvrir ce port MIDI en entrée"}
		cancel()
		return
	}
	logger.Debug("input port", "open", in.IsOpen())

	var absmillisec int32 = 0
	TICKS := smf.New().TimeFormat.(smf.MetricTicks)

	shoudStartRecording := false
	isRecording := false

	/*recordTrack.SendTo(TICKS, nil, func(m midi.Message, timestampms int32) {
		d := time.Duration(timestampms) * 10
		println(timestampms, d)
		send(m)
		time.Sleep(d)
	})*/
	playCtx, cancelPlay := context.WithCancel(ctx)
	playCtx = context.WithValue(playCtx, charmlog.ContextKey, logger)

	playTrack(playCtx, recordTrack, TICKS, send)
	time.Sleep(time.Second)
	playRTrack(playCtx, convert(recordTrack), TICKS, send)

	isSteps := false
	stepIndex := 0
	stepNotes := []uint8{}
	stop, err := midi.ListenTo(in, func(msg midi.Message, absms int32) {
		var ch, key, vel uint8

		switch {
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			on := msg.IsOneOf(midi.NoteOnMsg)
			//note := midi.Note(key)

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
				recordTrack = smf.Track{}
				recordTrack.Add(0, smf.MetaTempo(BPM))
				absmillisec = absms
				/*main.Add(0, msg)
				send(msg)
				return*/
			}
			if isRecording {
				deltams := absms - absmillisec
				absmillisec = absms
				delta := TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
				recordTrack.Add(delta, msg)
			}
			send(msg)
		case msg.GetControlChange(&ch, &key, &vel):
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
	currentlyPlaying := false
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
					recordTrack.Close(0)
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					SinkUI <- Message{ev: Record, boolean: true}
					logger.Debug("start recording")
				}
			case PlayPause:
				/*go func() {
					logger.Debug("start play")
					var bf bytes.Buffer
					tmpFile := smf.New()
					tmpFile.Add(recordTrack)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					player := smf.ReadTracksFrom(&bf)
					he(player.Play(out))
					SinkUI <- Message{ev: PlayPause}
					logger.Debug("end play")
				}()*/
				if currentlyPlaying {
					cancelPlay()
					logger.Info("stop playing")
				} else {
					playCtx, cancelPlay = context.WithCancel(ctx)
					//playCtx = context.WithValue(playCtx, charmlog.ContextKey, logger)
					context.AfterFunc(playCtx, func() {
						SinkUI <- Message{ev: PlayPause, boolean: false}
						currentlyPlaying = false
					})
					SinkUI <- Message{ev: PlayPause, boolean: true}
					go func() {
						currentlyPlaying = true
						logger.Info("start playing")
						//	playTrack(playCtx, recordTrack, TICKS, send)
						playRTrack(playCtx, convert(recordTrack), TICKS, send)
						logger.Info("finished playing")
						cancelPlay()
					}()
				}
			case Quantize:
				go func() {
					log.Printf("quantize at %d BPM\n", msg.number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					recordTrack[0].Message = smf.MetaTempo(float64(msg.number))
					tmpFile.Add(recordTrack)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					he(quantizer.Quantize(&bf, &bf))
					recordTrack = smf.ReadTracksFrom(&bf).SMF().Tracks[0]
					SinkUI <- Message{ev: Quantize}
					logger.Debug("quantize done")
				}()
			case StepMode:
				logger.Debug("set steps mode to", isSteps)
				isSteps = !isSteps
				stepIndex = 0
				SinkUI <- Message{ev: StepMode, boolean: isSteps}
				if isSteps {
					stepNotes = trackToSteps(recordTrack)
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
				recordTrack = midiFile.Tracks[0]
			case SaveToFile:
				fileName := msg.str
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				logger.Info("saving to", "filename", fileName)
				midiFile := smf.New()
				recordTrack.Close(0)
				if err := midiFile.Add(recordTrack); err != nil {
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

func saveTrack(tr smf.Track, filepath string) error {
	s := smf.New()
	s.Add(tr)
	return s.WriteFile(filepath)
}

func loadTrack(filepath string) (smf.Track, error) {
	s, err := smf.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	if s.NumTracks() > 0 {
		return s.Tracks[0], nil
	}
	return nil, errors.New("no tracks in file")
}

func playTrack(ctx context.Context, recordTrack smf.Track, ticks smf.MetricTicks, send func(midi.Message) error) error {
	absms := uint32(0)
	logger := charmlog.FromContext(ctx)
	var ch, key, vel uint8
	for _, ev := range recordTrack {
		absms += ev.Delta
		ev.Message.GetNoteOff(&ch, &key, &vel)
		if ev.Message.GetNoteOn(&ch, &key, &vel) {
			logger.Debug("note  on", "key", midi.Note(key), "delta", ev.Delta, "abs", absms)
		} else {
			logger.Debug("note off", "key", midi.Note(key), "delta", ev.Delta, "abs", absms)
		}
		if smf.Message(ev.Message).IsPlayable() {
			delta := ticks.Duration(BPM, ev.Delta)
			/*if ms < 0 {
				println(ms)
				continue
			}*/
			select {
			case <-ctx.Done():
				return nil
			default:
				time.Sleep(delta)
				if err := send(midi.Message(ev.Message)); err != nil {
					return err
				}
				//println(ev.Message.String(), delta.Milliseconds())
			}
		}
	}
	return nil
}
