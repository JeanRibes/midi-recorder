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

type RecTrack []RecEvent

func convert(tr smf.Track) RecTrack {
	rt := RecTrack{}
	/*
		noteOn(A) , noteOff(A) → OK
		noteOn(A), noteOff(B) → nope
	*/
	//synthTable := map[midi.Note]bool{}
	var ch, key, vel uint8
	prevOn := false     // previous message is noteOn
	prevKey := uint8(0) // previous NoteON
	prevVel := uint8(0)
	var on bool
	absTime := uint32(0)
	onAt := uint32(0)
	for _, ev := range tr {
		absTime += ev.Delta
		ev.Message.GetNoteOff(&ch, &key, &vel)
		on = ev.Message.GetNoteOn(&ch, &key, &vel)
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
		/*if on && prevOn { // empêche deux noteOn
			continue
		}
		if !on && key != prevKey { // empêche d'éteindre une autre note
			continue // que la précédente
		}
		prevOn = on
		prevKey = key
		if on {
			onAt = absTime
			prevVel = vel
		} else {
			rt = append(rt, RecEvent{
				note:     midi.Note(key),
				delta:    onAt,
				duration: onAt - absTime,
				vel:      prevVel,
			})
		}*/
	}
	println(len(rt))
	absTime = 0
	prevOn = true
	for i, ev := range rt[1:] {
		prev := rt[i]
		prev.silenceAfter = ev.delta - (prev.delta + prev.duration)
		rt[i] = prev
	}
	return rt
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
			absms += ev.delta
			logger.Debug("note  on", "key", ev.note, "delta", ev.delta, "duration", ev.duration, "abs", absms)
			send(midi.NoteOn(0, uint8(ev.note), ev.vel))
			time.Sleep(ticks.Duration(BPM, ev.duration))
			logger.Debug("note off", "key", ev.note, "delta", ev.delta, "silence", ev.silenceAfter, "abs", absms)
			send(midi.NoteOff(0, uint8(ev.note)))
			time.Sleep(ticks.Duration(BPM, ev.silenceAfter))
		}
	}
	return nil
}
