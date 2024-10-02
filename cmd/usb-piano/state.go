package main

import (
	"errors"
	"sync"

	"gitlab.com/gomidi/midi/v2/smf"
)

const NUM_BANKS = 6

type LoopState struct {
	banks     [NUM_BANKS]RecTrack // pour jouer en mode "steps"
	playBank  [NUM_BANKS]bool     // choisit les banques qui seront jouées
	tempTrack smf.Track           // pour enregistrer vers les banques
	stepIndex int
	sync.Mutex
}

const STATE_PREALLOCATION = 128

func NewState() *LoopState {
	state := LoopState{}
	state.tempTrack = make(smf.Track, 0, STATE_PREALLOCATION)
	for i := 0; i < NUM_BANKS; i++ {
		state.banks[i] = make(RecTrack, 0, STATE_PREALLOCATION)
	}
	return &state
}

func init() {
}

func (s *LoopState) RecBank(bank int, ev RecEvent) {
	s.banks[bank] = append(s.banks[bank], ev)
}

func (s *LoopState) LoadTrack(bank int, tr smf.Track) {
	s.Append(bank, convert(tr))
}

func (s *LoopState) Append(bank int, rt RecTrack) {
	s.Lock()
	s.banks[bank] = append(s.banks[bank], rt...)
	s.Unlock()
}

func (s *LoopState) Concat(dst, src int) {
	s.Append(dst, s.banks[src])
}

func (s *LoopState) TempIntoBank(bank int) {
	s.LoadTrack(bank, s.tempTrack)
}

func (s *LoopState) EndRecord() {
	s.TempIntoBank(0)
}

func (s *LoopState) Clear(bank int) {
	s.Lock()
	s.banks[bank] = s.banks[bank][0:0]
	s.Unlock()
}

/*
Joue les notes des banques actives, uniquement si elles sont assez longues pour contenir
l'index de la note
*/
func (s *LoopState) StepPlay() [NUM_BANKS]*RecEvent {
	res := [NUM_BANKS]*RecEvent{}

	s.Lock()
	for bankIndex, bank := range s.banks {
		if s.stepIndex+1 < len(bank) && s.playBank[bankIndex] {
			res[bankIndex] = &bank[s.stepIndex]
		}
	}
	s.Unlock()

	s.stepIndex += 1
	return res
}

func (s *LoopState) ResetStep() {
	s.stepIndex = 0
}

func (s *LoopState) EnableBank(bank int, enable bool) bool {
	s.playBank[bank] = enable
	return s.playBank[bank]
}

func (s *LoopState) SaveToFile(filepath string) (errs error) {
	f := smf.New()
	if err := f.Add(s.tempTrack); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, bank := range s.banks {
		if err := f.Add(bank.Convert()); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if err := f.WriteFile(filepath); err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

func (s *LoopState) LoadFromFile(filepath string) error {
	f, err := smf.ReadFile(filepath)
	if err != nil {
		return nil
	}
	if f.NumTracks() < 1 {
		return errors.New("no tracks in file")
	}
	s.tempTrack = f.Tracks[0]
	for i, track := range f.Tracks[1:] {
		s.Append(i, convert(track))
	}
	return nil
}
