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

func loop(ctx context.Context, cancel func(), in drivers.In, out drivers.Out, state *LoopState) {
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

	/*recordTrack.SendTo(TICKS, nil, func(m midi.Message, timestampms int32) {
	d := time.Duration(timestampms) * 10
	println(timestampms, d)
	send(m)
	time.Sleep(d)
	})*/
	playCtx, cancelPlay := context.WithCancel(ctx)
	playCtx = context.WithValue(playCtx, charmlog.ContextKey, logger)

	/*playTrack(playCtx, recordTrack, TICKS, send)
	time.Sleep(time.Second)
	playRTrack(playCtx, convert(recordTrack), TICKS, send)*/

	shoudStartRecording := false
	isRecording := false
	isSteps := false
	lastOnStep := [NUM_BANKS]*RecEvent{}
	stop, err := midi.ListenTo(in, func(msg midi.Message, absms int32) {
		var ch, key, vel uint8

		switch {
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			on := msg.IsOneOf(midi.NoteOnMsg)
			//note := midi.Note(key)

			if isSteps {
				if on {
					lastOnStep = state.StepPlay()
				}
				cnt := 0
				for _, ev := range lastOnStep {
					if ev != nil {
						send(ev.Message(on))
						cnt += 1
					}
				}
				if cnt == 0 {
					if on {
						send(midi.NoteOn(ch, 1, vel))
					} else {
						send(midi.NoteOff(ch, 1))
					}
				}
				return
			}

			if shoudStartRecording && on {
				// START record
				shoudStartRecording = false
				isRecording = true
				state.tempTrack = smf.Track{}
				state.tempTrack.Add(0, smf.MetaTempo(BPM))
				absmillisec = absms
				/*main.Add(0, msg)
				send(msg)
				return*/
			}
			if isRecording {
				deltams := absms - absmillisec
				absmillisec = absms
				delta := TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
				state.tempTrack.Add(delta, msg)
			}
			send(msg)
		case msg.GetControlChange(&ch, &key, &vel):
			if isSteps {
				state.ResetStep()
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

	for bank, leng := range state.Stats() {
		SinkUI <- Message{
			ev:     BankLengthNotify,
			number: bank,
			port2:  leng,
		}
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
					state.tempTrack.Close(0)
					state.Clear(0)
					state.EndRecord()
					logger.Debug("put recordtrack into bank 0", "len", len(state.banks[0]))
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
						playRTrack(playCtx, state.banks[0], TICKS, send)
						logger.Info("finished playing")
						cancelPlay()
					}()
				}
			case Quantize:
				go func() {
					log.Printf("quantize at %d BPM\n", msg.number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					state.tempTrack[0].Message = smf.MetaTempo(float64(msg.number))
					tmpFile.Add(state.tempTrack)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					he(quantizer.Quantize(&bf, &bf))
					state.LoadTrack(0, smf.ReadTracksFrom(&bf).SMF().Tracks[0])
					SinkUI <- Message{ev: Quantize}
					logger.Debug("quantize done")
				}()
			case StepMode:
				logger.Debug("set steps to", "mode", isSteps)
				isSteps = !isSteps
				SinkUI <- Message{ev: StepMode, boolean: isSteps}
				state.ResetStep()
				// load stuff
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
				state.tempTrack = midiFile.Tracks[0]
				state.Clear(0)
				state.EndRecord()
			case SaveToFile:
				fileName := msg.str
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				logger.Info("saving to", "filename", fileName)
				midiFile := smf.New()
				state.tempTrack.Close(0)
				if err := midiFile.Add(state.tempTrack); err != nil {
					logger.Error(err)
					continue
				}
				if err := midiFile.WriteFile(fileName); err != nil {
					logger.Error(err)
				}
			case BankStateChange:
				state.EnableBank(msg.number, msg.boolean)
				logger.Printf("set bank %d to state %t\n", msg.number, msg.boolean)
			case BankDragDrop:
				src := msg.number
				dst := msg.port2
				logger.Printf("append bank %d to bank %d\n", src, dst)

				l1 := len(state.banks[dst])
				state.Concat(dst, src)
				l2 := len(state.banks[dst])
				logger.Debug("bank %d went from", l1, l2)
				SinkUI <- Message{
					ev:     BankLengthNotify,
					number: dst,
					port2:  state.Stat(dst),
				}
			case BankClear:
				src := msg.number
				state.Clear(src)
				SinkUI <- Message{
					ev:     BankLengthNotify,
					number: src,
					port2:  state.Stat(src),
				}
			}
		}
	}
	logger.Info("stop")
	if out.IsOpen() {
		stop()
		//he(out.Close())
	} else {
		println("already closed")
	}
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
