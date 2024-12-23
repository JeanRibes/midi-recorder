package music

import (
	"context"
	"errors"
	"sync"

	. "github.com/JeanRibes/midi-recorder/shared"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
	"golang.org/x/sync/semaphore"
)

type LoopState struct {
	Banks     [NUM_BANKS]RecTrack // pour jouer en mode "steps"
	playBank  [NUM_BANKS]bool     // choisit les banques qui seront jouées
	TempTrack smf.Track           // pour enregistrer vers les banques
	StepIndex int
	MutiTrack smf.Track
	sync.Mutex
}

const STATE_PREALLOCATION = 128

func NewState() *LoopState {
	state := LoopState{}
	state.TempTrack = make(smf.Track, 0, STATE_PREALLOCATION)
	for i := 0; i < NUM_BANKS; i++ {
		state.Banks[i] = make(RecTrack, 0, STATE_PREALLOCATION)
	}
	return &state
}

func init() {
}

func (s *LoopState) RecBank(bank int, ev RecEvent) {
	s.Banks[bank] = append(s.Banks[bank], ev)
}

func (s *LoopState) LoadTrack(bank int, tr smf.Track) {
	s.Append(bank, Convert(tr))
}

func (s *LoopState) Append(bank int, rt RecTrack) {
	s.Lock()
	s.Banks[bank] = append(s.Banks[bank], rt...)
	s.Unlock()
}

func (s *LoopState) Concat(dst, src int) {
	s.Append(dst, s.Banks[src])
}

func (s *LoopState) TempIntoBank(bank int) {
	s.LoadTrack(bank, s.TempTrack)
}

func (s *LoopState) EndRecord() {
	s.TempIntoBank(0)
}

func (s *LoopState) Clear(bank int) {
	s.Lock()
	s.Banks[bank] = s.Banks[bank][0:0]
	s.Unlock()
}

/*
Joue les notes des banques actives, uniquement si elles sont assez longues pour contenir
l'index de la note
*/
func (s *LoopState) StepPlay() []*RecEvent {
	res := make([]*RecEvent, 0, NUM_BANKS)

	s.Lock()
	for bankIndex, bank := range s.Banks {
		if s.StepIndex < len(bank) && s.playBank[bankIndex] {
			res = append(res, &bank[s.StepIndex])
		}
	}
	s.Unlock()

	s.StepIndex += 1
	return res
}

func (s *LoopState) ResetStep() {
	s.StepIndex = 0
}

func (s *LoopState) StepBack() {
	if s.StepIndex > 0 {
		s.StepIndex -= 1
	}
}

func (s *LoopState) EnableBank(bank int, enable bool) bool {
	s.playBank[bank] = enable
	return s.playBank[bank]
}

func (s *LoopState) SaveToFile(filepath string) (errs error) {
	f := smf.New()
	f.TimeFormat = TICKS
	if err := f.Add(s.TempTrack); err != nil {
		errs = errors.Join(errs, err)
	}
	for _, bank := range s.Banks {
		if err := f.Add(bank.Convert()); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	f.Add(s.MutiTrack)
	/*f.Tracks[0][0] = smf.Event{
		Message: smf.MetaText("yo"),
		Delta:   0,
	}*/
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
	s.TempTrack = f.Tracks[0]
	for i, track := range f.Tracks[1:6] {
		s.Append(i, Convert(track))
	}
	s.MutiTrack = f.Tracks[f.NumTracks()-1]
	/*ms := ""
	if f.Tracks[0][0].Message.GetMetaText(&ms) {
		println(ms)
	}*/
	return nil
}

func (s *LoopState) Notify(sink chan Message) {
	for i, bank := range s.Banks {
		sink <- Message{
			Type:    BankLengthNotify,
			Number:  i,
			Number2: len(bank),
		}
	}
	return
}

func (s *LoopState) Stat(bank int) (res int) {
	s.Lock()
	res = len(s.Banks[bank])
	s.Unlock()
	return
}

/*
func init() {
	tr := smf.Track{}
	tr.Add(0, smf.MetaText("test"))
	smf.MetaText("test")
}
*/

func (s *LoopState) Play(ctx context.Context, send func(midi.Message) error) {
	numBanks := 0

	sem := semaphore.NewWeighted(NUM_BANKS)
	sem.Acquire(context.TODO(), NUM_BANKS)
	for i, enable := range s.playBank {
		if enable {
			go func() {
				println("play", i)
				s.Banks[i].Play(ctx, send)
				println("done")
				sem.Release(1)
				println("released", i)
			}()
			numBanks += 1
		}
	}
	if numBanks == 0 {
		PlayTrack(ctx, s.TempTrack, TICKS, send)
	} else {
		println("acquire")
		if err := sem.Acquire(ctx, int64(numBanks)); err != nil {
			println("erreur semathpre", err.Error())
		}
		println("ok")
	}
	println("finieshed play")
}

func (s *LoopState) ClearState() {
	s.Lock()
	s.TempTrack = smf.Track{}
	s.MutiTrack = smf.Track{}
	for i := range s.Banks {
		s.Banks[i] = RecTrack{}
	}
	s.Unlock()
}

func (s *LoopState) DeleteNote() {
	s.Lock()
	for bankIndex, bank := range s.Banks {
		if s.StepIndex < len(bank) && s.playBank[bankIndex] {
			//s.Banks[bankIndex] = slices.Delete(bank, s.StepIndex+1, s.StepIndex)
			if s.StepIndex == 0 {
				s.Banks[bankIndex] = bank[1:]
			} else {
				s.Banks[bankIndex] = append(bank[:s.StepIndex-1], bank[s.StepIndex:]...)
			}
		}
	}
	s.Unlock()
}

func (s *LoopState) Transpose(bank, shift int) {
	s.Lock()
	for i, ev := range s.Banks[bank] {
		note := int(ev.note) + shift
		if note < 0 {
			note = 1
		}
		if note > 127 {
			note = 127
		}
		ev.note = midi.Note(note)
		s.Banks[bank][i] = ev
	}
	s.Unlock()
}
