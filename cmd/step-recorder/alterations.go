package main

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
)

// permet de taper au piano sans se préoccuper des dièses et bémols

type Note int

const (
	Do  Note = 0
	Ré  Note = 2
	Mi  Note = 4
	Fa  Note = 5
	Sol Note = 7
	La  Note = 9
	Si  Note = 11
)

//go:generate stringer -type=Armure
type Armure int

const (
	DO_MAJEUR Armure = iota
	LA_MINEUR

	FA_MAJEUR
	RE_MINEUR

	SI_BEMOL_MAJEUR
	SOL_MINEUR

	MI_BEMOL_MAJEUR
	DO_MINEUR

	LA_BEMOL_MAJEUR
	FA_MINEUR

	RE_BEMOL_MAJEUR
	SI_BEMOL_MINEUR

	SOL_BEMOL_MAJEUR
	MI_BEMOL_MINEUR

	FA_DIESE_MAJEUR
	RE_DIESE_MINEUR

	SI_MAJEUR
	SOL_DIESE_MINEUR

	MI_MAJEUR
	DO_DIESE_MINEUR

	LA_MAJEUR
	FA_DIESE_MINEUR

	RE_MAJEUR
	SI_MINEUR

	SOL_MAJEUR
	MI_MINEUR
)

func midi_to_gamme_note(midi_note int) (int, Note) {
	note := Note(midi_note % 12)
	gamme := int(midi_note / 12)
	return gamme, note
}
func gamme_note_to_midi(gamme int, note Note) int {
	return gamme*12 + int(note)
}

type DemiTon int

const bémol DemiTon = -1
const dièse DemiTon = +1

func alter_gamme(x Note, shift DemiTon, notes ...Note) Note {
	for _, note := range notes {
		if x == note {
			return note + Note(shift)
		}
	}
	return x
}

func alter(midi_note int, alteration Armure) int {
	gamme, note := midi_to_gamme_note(midi_note)
	if gamme*12+int(note) != midi_note {
		panic(fmt.Sprintf("alter: math error: %d,%d,%d", midi_note, gamme, note))
	}

	switch alteration {
	case FA_MAJEUR, RE_MINEUR:
		switch note {
		case Si:
			return gamme_note_to_midi(gamme, note-1)
		}
	case SI_BEMOL_MAJEUR, SOL_MINEUR:
		switch note {
		case Si, Mi:
			return gamme_note_to_midi(gamme, note-1)
		}
	case MI_BEMOL_MAJEUR, DO_MINEUR:
		switch note {
		case La, Si, Mi:
			return gamme_note_to_midi(gamme, note-1)
		}
	case LA_BEMOL_MAJEUR, FA_MINEUR:
		switch note {
		case La, Si, Ré, Mi:
			return gamme_note_to_midi(gamme, note-1)
		}
	case RE_BEMOL_MAJEUR, SI_BEMOL_MINEUR:
		switch note {
		case Sol, La, Si, Ré, Mi:
			return gamme_note_to_midi(gamme, note-1)
		}
	case SOL_BEMOL_MAJEUR, MI_BEMOL_MINEUR:
		return gamme_note_to_midi(gamme,
			alter_gamme(note, bémol, Sol, La, Si, Do, Ré, Mi))
		/*switch note {
		case Sol, La, Si, Do, Ré, Mi:
			return gamme_note_to_midi(gamme, note-1)
		}*/
	case FA_DIESE_MAJEUR, RE_DIESE_MINEUR:
		switch note {
		case La, Do, Ré, Mi, Fa, Sol:
			return gamme_note_to_midi(gamme, note+1)
		}
	case SI_MAJEUR, SOL_DIESE_MINEUR:
		return gamme_note_to_midi(gamme, alter_gamme(note, dièse, La, Do, Ré, Fa, Sol))
	case MI_MAJEUR, DO_DIESE_MINEUR:
		return gamme_note_to_midi(gamme, alter_gamme(note, dièse, Do, Ré, Fa, Sol))
	case LA_MAJEUR, FA_DIESE_MINEUR:
		return gamme_note_to_midi(gamme, alter_gamme(note, dièse, Do, Fa, Sol))
	case RE_MAJEUR, SI_MINEUR:
		return gamme_note_to_midi(gamme, alter_gamme(note, dièse, Do, Fa))
	case SOL_MAJEUR, MI_MINEUR:
		return gamme_note_to_midi(gamme, alter_gamme(note, dièse, Fa))
	}

	return midi_note
}

func Alter(msg midi.Message, alteration Armure) midi.Message {
	var ch, key, vel uint8
	if msg.GetNoteOn(&ch, &key, &vel) {
		key = uint8(alter(int(key), alteration))
		return midi.NoteOn(ch, key, vel)
	}
	if msg.GetNoteEnd(&ch, &key) {
		key = uint8(alter(int(key), alteration))
		return midi.NoteOff(ch, key)
	}
	return msg
}
