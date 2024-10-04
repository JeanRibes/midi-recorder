package main

import (
	"context"
	"time"

	charmlog "github.com/charmbracelet/log"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

type RecEvent struct {
	note         midi.Note
	duration     uint32
	silenceAfter uint32
	delta        uint32
	vel          uint8
}

// func (ev *RecEvent) Play(send func(midi.Message) error) error {
func (ev *RecEvent) Message(on bool) midi.Message {
	if on {
		return midi.NoteOn(0, uint8(ev.note), ev.vel)
	} else {
		return midi.NoteOff(0, uint8(ev.note))
	}
}

type RecTrack []RecEvent

const DONT_SKIP_NOTES = true

func convert(tr smf.Track) RecTrack {
	rt := RecTrack{}
	if len(tr) <= 2 {
		return rt
	}
	/*
		noteOn(A) , noteOff(A) → OK
		noteOn(A), noteOff(B) → nope
		noteOn(A), noteOn(B) → on coupe A & on démarre B
	*/
	//synthTable := map[midi.Note]bool{}
	var ch, key, vel uint8
	prevOn := false     // previous message is noteOn
	prevKey := uint8(0) // previous NoteON
	prevVel := uint8(127)
	var on bool
	absTime := uint32(0)
	onAt := uint32(0)
	for _, ev := range tr {
		absTime += ev.Delta
		ev.Message.GetNoteOff(&ch, &key, &vel)
		on = ev.Message.GetNoteOn(&ch, &key, &vel)

		if DONT_SKIP_NOTES && on && prevOn { // commenter pour sauter les notes
			println(midi.Note(prevKey).String(), "→", midi.Note(key).String())
			rt = append(rt, RecEvent{
				note:         midi.Note(prevKey),
				delta:        onAt,
				duration:     absTime - onAt,
				silenceAfter: 1000,
				vel:          prevVel,
			})
			prevKey = key
			onAt = absTime
			continue
		}
		if !on && prevOn && prevKey == key {
			rt = append(rt, RecEvent{
				note:     midi.Note(key),
				delta:    onAt,
				duration: absTime - onAt,
				vel:      prevVel,
			})
			prevOn = false
			continue
		}
		if !prevOn && on {
			onAt = absTime
			prevOn = on
			prevVel = vel
			prevKey = key
		}
	}
	absTime = 0
	prevOn = true
	for i, ev := range rt[1:] {
		prev := rt[i]
		prev.silenceAfter = ev.delta - (prev.delta + prev.duration)
		rt[i] = prev
	}
	return rt
}

func (rt *RecTrack) Convert() smf.Track {
	tr := smf.Track{}
	tr.Add(0, smf.MetaTempo(BPM))
	absTicks := uint32(0)
	for _, ev := range *rt {
		absTicks += ev.delta
		tr.Add(absTicks, ev.Message(true))
		absTicks += ev.duration
		tr.Add(absTicks, ev.Message(false))
		absTicks += ev.silenceAfter
	}
	tr.Close(absTicks)
	return tr
}

func playRTrack(ctx context.Context, recordTrack RecTrack, ticks smf.MetricTicks, send func(midi.Message) error) error {
	absms := uint32(0)
	logger := charmlog.FromContext(ctx)
	logger.Info("play RT")
	for _, ev := range recordTrack {
		select {
		case <-ctx.Done():
			return nil
		default:
			absms += ev.duration
			logger.Debug("note  on", "key", ev.note, "delta", ev.delta, "duration", ev.duration, "abs", absms)
			send(midi.NoteOn(0, uint8(ev.note), ev.vel))
			time.Sleep(ticks.Duration(BPM, ev.duration))

			absms += ev.silenceAfter
			logger.Debug("note off", "key", ev.note, "delta", ev.delta, "silence", ev.silenceAfter, "abs", absms)
			send(midi.NoteOff(0, uint8(ev.note)))
			time.Sleep(ticks.Duration(BPM, ev.silenceAfter))
		}
	}
	return nil
}