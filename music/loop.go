package music

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	. "github.com/JeanRibes/midi-recorder/shared"

	"github.com/charmbracelet/log"
	charmlog "github.com/charmbracelet/log"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/smf"
	"gitlab.com/gomidi/quantizer/lib/quantizer"
)

func Run(ctx context.Context, cancel func(), in drivers.In, out drivers.Out, state *LoopState, SinkUI, SinkLoop, MasterControl chan Message) {
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

	uiAlert := func(str string) {
		SinkUI <- Message{Type: Error, String: str}
	}
	uiError := func(err error) {
		uiAlert(err.Error())
	}

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
	lastOnStep := []*RecEvent{}
	stop, err := midi.ListenTo(in, func(msg midi.Message, absms int32) {
		var ch, key, vel uint8
		var delta uint32
		switch {
		case msg.GetNoteOn(&ch, &key, &vel) || msg.GetNoteEnd(&ch, &key):
			if isRecording {
				deltams := absms - absmillisec
				absmillisec = absms
				delta = TICKS.Ticks(BPM, time.Duration(deltams)*time.Millisecond)
			}
			on := msg.IsOneOf(midi.NoteOnMsg)

			if shoudStartRecording && on {
				// START record
				shoudStartRecording = false
				isRecording = true
				absmillisec = absms
				if isSteps {
					state.MutiTrack = smf.Track{}
					state.MutiTrack.Add(0, smf.MetaTempo(BPM))
				} else {
					logger.Debug("started recording multi-track")
					state.TempTrack = smf.Track{}
					state.TempTrack.Add(0, smf.MetaTempo(BPM))
					//state.TempTrack.Add(0, msg)
				}
			}

			if isSteps {
				if on {
					lastOnStep = state.StepPlay()
				}
				cnt := 0
				for _, ev := range lastOnStep {
					smsg := ev.MessageVel(on, vel)
					send(smsg)
					if isRecording {
						state.MutiTrack.Add(delta, smsg)
						delta = 0
					}
					cnt += 1
				}
				if cnt == 0 {
					if on {
						send(midi.NoteOn(ch, 1, vel))
					}
					if isRecording {
						return
					}
					logger.Info("steps finished")
					state.ResetStep()
					isSteps = false
					SinkUI <- Message{Type: StepMode, Boolean: isSteps}
				}
				return
			} else {
				if isRecording {
					state.TempTrack.Add(delta, msg)
				}
			}

			send(msg)
		case msg.GetControlChange(&ch, &key, &vel):
			if vel != 127 {
				return
			}
			if isSteps {
				if state.StepIndex == 0 {
					isSteps = false
					SinkUI <- Message{Type: StepMode, Boolean: isSteps}
				}
				state.ResetStep()
			} else {
				SinkUI <- Message{Type: StepMode, Boolean: true}
			}
		}
	})
	if err != nil {
		logger.Error(err)
		SinkUI <- Message{Type: Error, String: "impossible d'écouter ce port MIDI"}
		cancel()
		return
	}

	state.Notify(SinkUI)

	buffered_send := Scheduler(send) // pour les erreurs MIDI

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
					if isSteps {
						state.MutiTrack.Close(0)
					} else {
						state.TempTrack.Close(0)
						state.EndRecord()
						logger.Debug("put recordtrack into bank 0", "len", len(state.Banks[0]))
					}
					SinkUI <- Message{
						Type:    BankLengthNotify,
						Number:  0,
						Number2: state.Stat(0),
					}
				} else { //START RECORD
					shoudStartRecording = true
					isRecording = false
					SinkUI <- Message{Type: Record, Boolean: true}
					logger.Debug("start recording")
				}
			case PlayPause:
				if currentlyPlaying {
					cancelPlay()
					logger.Info("stop playing")
				} else {
					playCtx, cancelPlay = context.WithCancel(ctx)
					//playCtx = context.WithValue(playCtx, charmlog.ContextKey, logger)
					context.AfterFunc(playCtx, func() {
						SinkUI <- Message{Type: PlayPause, Boolean: false}
						currentlyPlaying = false
						logger.Debug("play afterfunc")
					})
					SinkUI <- Message{Type: PlayPause, Boolean: true}
					currentlyPlaying = true
					logger.Info("start playing")
					//PlayTrack(playCtx, state.TempTrack, TICKS, send)
					go func() {
						if isSteps {
							log.Info("start play multi-track")
							if err := PlayTrack(playCtx, state.MutiTrack, TICKS, buffered_send); err != nil {
								logger.Error(err)
							}
							log.Info("stop play multi-track")
						} else {
							state.Play(playCtx, buffered_send)
						}
						//buffered_send(state.Banks[0][0].Message(true))
						logger.Info("finished playing")
						currentlyPlaying = false
						cancelPlay()
					}()
					logger.Debug("play queued")
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
				isSteps = msg.Boolean
				logger.Debug("set steps to", "mode", isSteps)
				SinkUI <- Message{Type: StepMode, Boolean: isSteps}
				state.ResetStep()
			case ResetStep:
				state.ResetStep()
			case StepBack:
				state.StepBack()
			case StateImport:
				logger.Debug("loading state", "file", msg.String)
				if err := state.LoadFromFile(msg.String); err != nil {
					logger.Error(err)
					uiError(err)
				}
				state.Notify(SinkUI)
			case StateExport:
				fileName := msg.String
				if !strings.HasSuffix(fileName, ".mid") {
					fileName += ".mid"
				}
				logger.Info("saving to", "filename", fileName)
				if err := state.SaveToFile(fileName); err != nil {
					logger.Error(err)
					uiError(err)
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
				if !strings.HasSuffix(filepath, ".mid") {
					filepath += ".mid"
				}
				state.Lock()
				track := state.Banks[src].Convert()
				state.Unlock()
				f := smf.New()
				f.Add(track)
				if err := f.WriteFile(filepath); err != nil {
					logger.Error(err)
					uiError(err)
				}
			case BankImport:
				dst := msg.Number
				filepath := msg.String
				tr := smf.ReadTracks(filepath, 1)
				if tr == nil {
					uiAlert("impossible de lire le fichier " + filepath)
					continue
				}
				file := tr.SMF()
				if file == nil {
					uiAlert("impossible de décoder le fichier " + filepath)
					continue
				}
				tracks := file.Tracks
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
				if state.StepIndex >= len(bank) {
					state.Unlock()
					continue
				}
				cut := bank[:state.StepIndex]
				state.Unlock()
				state.Append(0, cut)
				SinkUI <- Message{
					Type:    BankLengthNotify,
					Number:  0,
					Number2: state.Stat(0),
				}
				logger.Info("cut bank", "bank", src, "len", len(cut))
			case NoteUndo:
				if isRecording {
					l := len(state.TempTrack)
					if l > 2 {
						state.TempTrack = state.TempTrack[0 : l-2] // on supprimes les 2 derniers messages: noteOn & noteOff
					}
					logger.Info("suppression de la dernière note", "avant", l, "après", len(state.TempTrack))
				} else {
					state.DeleteNote()
					state.Notify(SinkUI)
				}
			case ExportMultiTrack:
				if !state.MutiTrack.IsClosed() {
					logger.Warn("multi-track was not closed")
					state.MutiTrack.Close(0)
				}
				filepath := msg.String
				if !strings.HasSuffix(filepath, ".mid") {
					filepath += ".mid"
				}
				f := smf.New()
				f.Add(state.MutiTrack)
				if err := f.WriteFile(filepath); err != nil {
					logger.Error(err)
					SinkLoop <- Message{
						Type:   Error,
						String: err.Error(),
					}
				}
			case ClearState:
				state.ClearState()
				state.Notify(SinkUI)
			case Transpose:
				state.Transpose(msg.Number, msg.Number2)
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
