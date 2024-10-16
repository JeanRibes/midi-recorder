package ui

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"time"
)

type RecentFile struct {
	Path string `json:"path"`
	Time int64  `json:"time"`
}
type RecentFiles []RecentFile
type Preferences struct {
	RecentSessions RecentFiles `json:"recent_sessions"`
	RecentTracks   RecentFiles `json:"recent_tracks"`
}

func LoadPreferences() (*Preferences, error) {
	f, err := os.Open("data.json")
	prefs := &Preferences{
		RecentSessions: RecentFiles{},
		RecentTracks:   RecentFiles{},
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
	go prefs.Refresh()
	return prefs, nil
}

func hash(s string) int {
	num := 0
	for _, char := range []byte(s) {
		num += int(char)
	}
	return num
}

func unique(sl RecentFiles) RecentFiles {
	counter := map[int]int{}
	slices.Reverse(sl)
	for _, rf := range sl {
		if count, ok := counter[hash(rf.Path)]; ok {
			counter[hash(rf.Path)] = count + 1
		} else {
			counter[hash(rf.Path)] = 1
		}
	}
	out := RecentFiles{}
	for _, rf := range sl {
		if counter[hash(rf.Path)] > 0 {
			out = append(out, rf)
			counter[hash(rf.Path)] = 0
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
	p.RecentTracks = unique(append(p.RecentTracks, RecentFile{
		Path: path,
		Time: time.Now().Unix(),
	}))
}

func (p *Preferences) AddSession(path string) {
	p.RecentSessions = unique(append(p.RecentSessions, RecentFile{
		Path: path,
		Time: time.Now().Unix(),
	}))
}

func (p *Preferences) Sessions() RecentFiles {
	s := slices.Clone(p.RecentSessions)
	slices.Reverse(s)
	return s
}

func (p *Preferences) Tracks() RecentFiles {
	s := slices.Clone(p.RecentTracks)
	slices.Reverse(s)
	return s
}

func (p *Preferences) Refresh() {
	for i, item := range p.RecentTracks {
		stat, err := os.Stat(item.Path)
		if err != nil {
			continue
		}
		item.Time = stat.ModTime().Unix()
		p.RecentTracks[i] = item
	}

	for i, item := range p.RecentSessions {
		stat, err := os.Stat(item.Path)
		if err != nil {
			continue
		}
		item.Time = stat.ModTime().Unix()
		p.RecentSessions[i] = item
	}
}

func (_rfs *RecentFiles) LCP() string {
	rfs := *_rfs
	if len(rfs) <= 1 {
		return ""
	}
	prefix := rfs[0].Path

	// Compare the prefix with the other strings
	for _, rf := range rfs {
		for !strings.HasPrefix(rf.Path, prefix) {
			// Progressively reduce the prefix until it matches
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func (p *Preferences) DeleteTrack(filepath string) {
	p.RecentTracks = slices.DeleteFunc(p.RecentTracks, func(e RecentFile) bool {
		if e.Path == filepath {
			return true
		}
		return false
	})
}
