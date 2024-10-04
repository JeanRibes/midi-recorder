package music

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"

	. "github.com/JeanRibes/midi/shared"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

func Run(ctx context.Context, cancel func(), in drivers.In, out drivers.Out, state *LoopState) {
	LoopDied = false
	logger := charmlog.NewWithOptions(os.Stdout, charmlog.Options{
		Level:           charmlog.DebugLevel,
		ReportCaller:    true,
		ReportTimestamp: false,
		Prefix:          "loop",
	})
	logger.Info("start")
	logger.Info("connecting to", "input", in.String())
	logger.Info("connecting to", "output", out.String())

	send, err := midi.SendTo(out)
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{Type: Error, String: "impossible d'ouvrir ce port MIDI en sortie"}
		cancel()
		return
	}
	if err := send(midi.Reset()); err != nil {
		logger.Error(err)
	}
	// sustain
	if err := send(midi.ControlChange(0, 64, 127)); err != nil {
		logger.Error(err)
	}
	if in == nil {
		logger.Error("input port is nil")
		SinkUI <- Message{Type: Error, String: "impossible d'ouvrir ce port MIDI en entrée"}
		cancel()
		return
	}
	logger.Debug("input port", "open", in.IsOpen())

	var absmillisec int32 = 0

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
				state.TempTrack = smf.Track{}
				state.TempTrack.Add(0, smf.MetaTempo(BPM))
				absmillisec = absms
				/*main.Add(0, msg)
				send(msg)
				return*/
			}
			if isRecording {
				deltams := absms - absmillisec
				absmillisec = absms
				delta := TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
				state.TempTrack.Add(delta, msg)
			}
			send(msg)
		case msg.GetControlChange(&ch, &key, &vel):
			if isSteps {
				state.ResetStep()
			} else {
				if vel == 127 {
					SinkLoop <- Message{Type: Record}
				}
			}
		}
	})
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{Type: Error, String: "impossible d'écouter ce port MIDI"}
		cancel()
		return
	}

	for bank, leng := range state.Stats() {
		SinkUI <- Message{
			Type:    BankLengthNotify,
			Number:  bank,
			Number2: leng,
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
			switch msg.Type {
			case Record:
				logger.Debug("record SYN")
				if shoudStartRecording && !isRecording {
					shoudStartRecording = false
					SinkUI <- Message{Type: Record, Boolean: false}
					logger.Debug("cancel recording")
					continue
				}
				// time.Sleep(time.Second)
				if isRecording { // STOP RECORD
					isRecording = false
					SinkUI <- Message{Type: Record, Boolean: false}
					logger.Debug("stop recording")
					state.TempTrack.Close(0)
					state.Clear(0)
					state.EndRecord()
					logger.Debug("put recordtrack into bank 0", "len", len(state.Banks[0]))
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					SinkUI <- Message{Type: Record, Boolean: true}
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
						SinkUI <- Message{Type: PlayPause, Boolean: false}
						currentlyPlaying = false
					})
					SinkUI <- Message{Type: PlayPause, Boolean: true}
					go func() {
						currentlyPlaying = true
						logger.Info("start playing")
						//	playTrack(playCtx, recordTrack, TICKS, send)
						//PlayRTrack(playCtx, state.banks[0], TICKS, send)
						PlayTrack(playCtx, state.TempTrack, TICKS, send)
						logger.Info("finished playing")
						cancelPlay()
					}()
				}
			case Quantize:
				go func() {
					logger.Printf("quantize at %d BPM", msg.Number)
					var bf bytes.Buffer
					tmpFile := smf.New()
					state.TempTrack[0].Message = smf.MetaTempo(float64(msg.Number))
					tmpFile.Add(state.TempTrack)
					_, err = tmpFile.WriteTo(&bf)
					if err != nil {
						logger.Error(err)
						return
					}
					if err := quantizer.Quantize(&bf, &bf); err != nil {
						logger.Error(err)
						return
					}
					state.LoadTrack(0, smf.ReadTracksFrom(&bf).SMF().Tracks[0])
					SinkUI <- Message{Type: Quantize}
					logger.Debug("quantize done")
				}()
			case StepMode:
				logger.Debug("set steps to", "mode", isSteps)
				isSteps = !isSteps
				SinkUI <- Message{Type: StepMode, Boolean: isSteps}
				state.ResetStep()
			case ResetStep:
				state.ResetStep()
				// load stuff
			case StateImport:
				logger.Debug("loading state", "file", msg.String)
				if err := state.LoadFromFile(msg.String); err != nil {
					logger.Error(err)
					SinkUI <- Message{Type: Error, String: err.Error()}
				}
			case StateExport:
				fileName := msg.String
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				logger.Info("saving to", "filename", fileName)
				if err := state.SaveToFile(fileName); err != nil {
					logger.Error(err)
					SinkUI <- Message{Type: Error, String: err.Error()}
				}
			case BankStateChange:
				state.EnableBank(msg.Number, msg.Boolean)
				logger.Printf("set bank %d to state %t", msg.Number, msg.Boolean)
			case BankDragDrop:
				src := msg.Number
				dst := msg.Number2
				if src >= NUM_BANKS || dst >= NUM_BANKS {
					logger.Warn("tried to append to/from non-existent bank", "src", src, "dst", dst)
					continue
				}
				logger.Printf("append bank %d to bank %d", src, dst)

				l1 := len(state.Banks[dst])
				state.Concat(dst, src)
				l2 := len(state.Banks[dst])
				logger.Debug("bank %d went from", l1, l2)
				SinkUI <- Message{
					Type:    BankLengthNotify,
					Number:  dst,
					Number2: state.Stat(dst),
				}
				if dst == 0 {
					state.Lock()
					state.TempTrack = state.Banks[0].Convert()
					state.Unlock()
				}
			case BankClear:
				src := msg.Number
				if src >= NUM_BANKS {
					logger.Warn("tried to delete non-existent bank", "bank", src)
					continue
				}
				state.Clear(src)
				SinkUI <- Message{
					Type:    BankLengthNotify,
					Number:  src,
					Number2: state.Stat(src),
				}
			case BankExport:
				src := msg.Number
				filepath := msg.String
				state.Lock()
				track := state.Banks[src].Convert()
				state.Unlock()
				f := smf.New()
				f.Add(track)
				if err := f.WriteFile(filepath); err != nil {
					logger.Error(err)
					SinkLoop <- Message{
						Type:   Error,
						String: err.Error(),
					}
				}
			case BankImport:
				dst := msg.Number
				filepath := msg.String
				tr := smf.ReadTracks(filepath, 1)
				tracks := tr.SMF().Tracks
				if len(tracks) > 0 {
					state.LoadTrack(dst, tracks[0])
				}
				SinkUI <- Message{
					Type:    BankLengthNotify,
					Number:  dst,
					Number2: state.Stat(dst),
				}
			case BankCut:
				src := msg.Number
				if src >= NUM_BANKS {
					logger.Warn("tried to cut non-existent bank", "bank", src)
					continue
				}
				state.Lock()
				bank := state.Banks[src]
				cut := bank[state.StepIndex:]
				state.Unlock()
				state.Append(0, cut)
				SinkUI <- Message{
					Type:    BankLengthNotify,
					Number:  0,
					Number2: state.Stat(0),
				}
				logger.Info("cut bank", "bank", src, "len", len(cut))
			default:
				logger.Printf("unknown message type: %#v", msg.Type)
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
