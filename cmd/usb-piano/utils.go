package main

import (
	"github.com/gotk3/gotk3/gtk"
	"gitlab.com/gomidi/midi/v2/smf"
)

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

func targ(t *gtk.TargetEntry, err error) gtk.TargetEntry {
	return *t
}
