package ui

import (
	"encoding/json"
	"os"
	"slices"
)

type Preferences struct {
	RecentSessions []string `json:"recent_sessions"`
	RecentTracks   []string `json:"recent_tracks"`
}

func LoadPreferences() (*Preferences, error) {
	f, err := os.Open("data.json")
	prefs := &Preferences{
		RecentSessions: []string{},
		RecentTracks:   []string{},
	}
	if err != nil {
		return prefs, err
	}
	stat, err := f.Stat()
	if err != nil {
		return prefs, err
	}
	if stat.Size() < 10 {
		return prefs, nil
	}
	if err := json.NewDecoder(f).Decode(prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}

func hash(s string) int {
	num := 0
	for _, char := range []byte(s) {
		num += int(char)
	}
	return num
}

func unique(sl []string) []string {
	counter := map[int]int{}
	slices.Reverse(sl)
	for _, str := range sl {
		if count, ok := counter[hash(str)]; ok {
			counter[hash(str)] = count + 1
		} else {
			counter[hash(str)] = 1
		}
	}
	out := []string{}
	for _, str := range sl {
		if counter[hash(str)] > 0 {
			out = append(out, str)
			counter[hash(str)] = 0
		}
	}
	slices.Reverse(out)
	return out
}

func (p *Preferences) Save() error {
	f, err := os.Create("data.json")
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(p); err != nil {
		return err
	}
	println("wrote prefs")
	return f.Close()
}

func (p *Preferences) AddTrack(path string) {
	p.RecentTracks = unique(append(p.RecentTracks, path))
}

func (p *Preferences) AddSession(path string) {
	p.RecentSessions = unique(append(p.RecentSessions, path))
}

func (p *Preferences) Sessions() []string {
	s := slices.Clone(p.RecentSessions)
	slices.Reverse(s)
	return s
}

func (p *Preferences) Tracks() []string {
	s := slices.Clone(p.RecentTracks)
	slices.Reverse(s)
	return s
}
