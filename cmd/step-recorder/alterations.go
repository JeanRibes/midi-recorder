package main

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2"
)

type Armure string

// permet de taper au piano sans se préoccuper des dièses et bémols
type Note int

//go:generate stringer -type=Note
const (
	SiDièse  Note = 0 //octave +1
	Do       Note = 0
	DoDièse  Note = 1
	RéBémol  Note = 1
	Ré       Note = 2
	RéDièse  Note = 3
	MiBémol  Note = 3
	Mi       Note = 4
	Fa       Note = 5
	FaDièse  Note = 6
	SolBémol Note = 6
	Sol      Note = 7
	SolDièse Note = 8
	LaBémol  Note = 8
	La       Note = 9
	SiBémol  Note = 10
	Si       Note = 11
	DoBémol  Note = 11 //octave -1
)

type DemiTon int

//go:generate stringer -type=DemiTon
const (
	bémol DemiTon = -1
	dièse DemiTon = +1
)

/*const bémol DemiTon = -1
const dièse DemiTon = +1*/

func (dt DemiTon) Opposé() DemiTon {
	return -1 * dt
}

func (note Note) Shift(demiton DemiTon) Note {
	note = note + Note(demiton)

	return note
}
func (note Note) Clean() Note {
	if note > Si {
		note = Do
	}
	if note < Do {
		note = Si
	}
	return note
}

type Gamme struct {
	Becarres   []Note //généré, reverse map
	Altération DemiTon
	Notes      []Note
}

var Gammes map[Armure]Gamme
var GammesNames []Armure

func (armure Armure) ToString() string {
	gamme, ok := Gammes[armure]
	if !ok {
		return "erreur!"
	}
	return fmt.Sprintf("%s (%d %s)", armure, len(gamme.Notes), gamme.Altération.String())
}

func fillGammes() {
	Gammes = map[Armure]Gamme{
		"Do Majeur":       {[]Note{}, bémol, []Note{}},
		"Fa Majeur":       {[]Note{}, bémol, []Note{Si}},
		"SiBémol Majeur":  {[]Note{}, bémol, []Note{Si, Mi}},
		"MiBémol Majeur":  {[]Note{}, bémol, []Note{Si, Mi, La}},
		"LaBémol Majeur":  {[]Note{}, bémol, []Note{Si, Mi, La, Ré}},
		"RéBémol Majeur":  {[]Note{}, bémol, []Note{Si, Mi, La, Ré, Sol}},
		"SolBémol Majeur": {[]Note{}, bémol, []Note{Si, Mi, La, Ré, Sol, Do}},
		"FaDièse Majeur":  {[]Note{}, dièse, []Note{Fa, Do, Sol, Ré, La, Fa}},
		"Si Majeur":       {[]Note{}, dièse, []Note{Fa, Do, Sol, Ré, La}},
		"Mi Majeur":       {[]Note{}, dièse, []Note{Fa, Do, Sol, Ré}},
		"La Majeur":       {[]Note{}, dièse, []Note{Fa, Do, Sol}},
		"Ré Majeur":       {[]Note{}, dièse, []Note{Fa, Do}},
		"Sol Majeur":      {[]Note{}, dièse, []Note{Fa}},
	}
	for i, gamme := range Gammes {
		for _, note := range gamme.Notes {
			gamme.Becarres = append(gamme.Becarres, note.Shift(gamme.Altération).Clean())
			Gammes[i] = gamme
		}
	}
	/*Gammes["La Mineur"] = Gammes["Do Majeur"]
	Gammes["Ré Mineur"] = Gammes["Fa Majeur"]
	Gammes["Sol Mineur"] = Gammes["SiBémol Majeur"]
	Gammes["Do Mineur"] = Gammes["MiBémol Majeur"]
	Gammes["Fa Mineur"] = Gammes["LaBémol Majeur"]
	Gammes["SiBémo Mineur"] = Gammes["RéBémol Majeur"]
	Gammes["MiBémol Mineur"] = Gammes["SolBémol Majeur"]
	Gammes["RéDièse Mineur"] = Gammes["FaDièse Majeur"]
	Gammes["SolDièse Mineur"] = Gammes["Si Majeur"]
	Gammes["DoDièse Mineur"] = Gammes["Mi Majeur"]
	Gammes["FaDièse Mineur"] = Gammes["La Majeur"]
	Gammes["Si Mineur"] = Gammes["Ré Majeur"]
	Gammes["Mi Mineur"] = Gammes["Sol Majeur"]*/

	GammesNames = []Armure{
		"Do Majeur",
		//"La Mineur",
		"Fa Majeur",
		//"Ré Mineur",
		"SiBémol Majeur",
		//"Sol Mineur",
		"MiBémol Majeur",
		//"Do Mineur",
		"LaBémol Majeur",
		//"Fa Mineur",
		"RéBémol Majeur",
		//"SiBémo Mineur",
		"SolBémol Majeur",
		//"MiBémol Mineur",
		"FaDièse Majeur",
		//"RéDièse Mineur",
		"Si Majeur",
		//	"SolDièse Mineur",
		"Mi Majeur",
		//	"DoDièse Mineur",
		"La Majeur",
		//"FaDièse Mineur",
		"Ré Majeur",
		//	"Si Mineur",
		"Sol Majeur",
		//"Mi Mineur",
	}
	for key, gamme := range Gammes {
		fmt.Printf("%s %#v %#v, %s\n", key, gamme.Notes, gamme.Becarres, gamme.Altération.String())
	}
}

func init() {
	fillGammes()
}

func (g Gamme) alter(octave int, source Note) (int, Note) {
	for _, note := range g.Notes {
		if source == note {
			altered := source.Shift(g.Altération)
			fmt.Println("alteration:", note.String(), altered.String())
			return octave, altered
		}
	}
	for _, note := range g.Becarres { // pour les bécarres
		if source == note {
			becarre := source.Shift(g.Altération.Opposé()).Clean()
			fmt.Println("becarre:", note.String(), becarre.String())
			if note == Do && g.Altération == bémol {
				return octave - 1, becarre
			}
			if note == Si && g.Altération == dièse {
				return octave + 1, becarre
			}
			return octave, becarre
		}
	}
	return octave, source
}

var Mineurs map[string]Gamme

func midi_to_gamme_note(midi_note int) (int, Note) {
	note := Note(midi_note % 12)
	gamme := int(midi_note / 12)
	return gamme, note
}
func gamme_note_to_midi(gamme int, note Note) int {
	if note > Si {
		note = Do
		gamme += 1
	}
	if note < Do {
		note = Si
		gamme -= 1
	}
	return gamme*12 + int(note)
}

func alter_gamme(x Note, shift DemiTon, notes ...Note) Note {
	for _, note := range notes {
		if x == note {
			return note + Note(shift)
		}
	}
	return x
}

func alter(midi_note int, alteration Armure) int {
	octave, note := midi_to_gamme_note(midi_note)
	if octave*12+int(note) != midi_note {
		panic(fmt.Sprintf("alter: math error: %d,%d,%d", midi_note, octave, note))
	}
	gamme := Gammes[alteration]
	octave, note = gamme.alter(octave, note)
	return gamme_note_to_midi(octave, note)
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
