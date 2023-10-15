package main

import (
	"fmt"
	"time"

	"gitlab.com/gomidi/midi/v2"
)

type NoteOnOff struct {
	Note     uint8
	Duration time.Duration
	Wait     time.Duration // silence avant la note
}

func (ev NoteOnOff) Play(send func(midi.Message) error) {
	fmt.Printf("play #%d\n", ev.Note)
	send(midi.NoteOn(0, ev.Note, 64))
	time.Sleep(ev.Duration)
	send(midi.NoteOff(0, ev.Note))
}

type SingleRecording []NoteOnOff

// Filter chords so that only one note is ever playing at a given time
// if we detect that a note is already being played, all others will be ignored
func (r Recording) RemoveChords2() SingleRecording {
	out := SingleRecording{}
	previous_note := uint8(128)
	abs_time := time.Unix(0, 0)
	previous_time := time.Unix(0, 0)
	silence_start := time.Unix(0, 0)
	silence := time.Duration(0)

	for _, ev := range r {
		var ch, key, vel uint8
		var on bool
		if ev.Msg.GetNoteOn(&ch, &key, &vel) {
			abs_time = abs_time.Add(ev.Time)
			on = true
		}
		if ev.Msg.GetNoteEnd(&ch, &key) {
			abs_time = abs_time.Add(ev.Time)
			on = false
		}
		if on && previous_note < 128 { // note already playing, skip
			continue
		}
		if !on && previous_note == key {
			out = append(out, NoteOnOff{
				Note:     previous_note,
				Duration: abs_time.Sub(previous_time),
				Wait:     silence,
			})
			previous_note = 128
			silence_start = abs_time
		}
		if on {
			previous_note = key
			previous_time = abs_time
			silence = abs_time.Sub(silence_start)
		}
	}
	return out
}

func (r SingleRecording) Play(send func(msg midi.Message) error) {
	for _, ev := range r {
		time.Sleep(ev.Wait)
		fmt.Printf("note %d for %d ms, waited %d ms\n", ev.Note, ev.Duration.Milliseconds(), ev.Wait.Milliseconds())
		send(midi.NoteOn(0, ev.Note, 64))
		time.Sleep(ev.Duration)
		send(midi.NoteOff(0, ev.Note))
	}
}

// Filter chords so that only one note is ever playing at a given time
// if we detect that a note is already being played, it will be stopped
func (r Recording) RemoveChords1() SingleRecording {
	out := SingleRecording{}
	previous_note := uint8(128)
	abs_time := time.Unix(0, 0)
	previous_time := time.Unix(0, 0)

	for _, ev := range r {
		var ch, key, vel uint8
		var on bool
		if ev.Msg.GetNoteOn(&ch, &key, &vel) {
			abs_time = abs_time.Add(ev.Time)
			on = true
		}
		if ev.Msg.GetNoteEnd(&ch, &key) {
			abs_time = abs_time.Add(ev.Time)
			on = false
		}
		if on {
			if previous_note < 128 { // note already playing, stop it
				out = append(out, NoteOnOff{
					Note:     previous_note,
					Duration: abs_time.Sub(previous_time),
				})
				previous_note = key
			} else {
				previous_note = key
			}
		}
		if !on && previous_note < 128 {
			out = append(out, NoteOnOff{
				Note:     key,
				Duration: abs_time.Sub(previous_time),
			})
			previous_note = 128
		}
		previous_time = abs_time
	}
	return out
}
