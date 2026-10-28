package model

import (
	"math/rand"
	"time"
)

type Playlist struct {
	tracks          []Track
	current         int
	shuffle         bool
	repeat          RepeatMode
	shuffleOrder    []int
	shuffleIdx      int
	shuffleStartIdx int
	revision        uint64
}

func NewPlaylist() *Playlist {
	return &Playlist{tracks: make([]Track, 0), current: -1}
}

func (p *Playlist) Add(track Track) {
	p.tracks = append(p.tracks, track)
	p.revision++
	if p.current == -1 {
		p.current = 0
	}
	if p.shuffle {
		p.regenerateShuffle()
	}
}

func (p *Playlist) Remove(index int) {
	if !p.isValidIndex(index) {
		return
	}
	p.tracks = append(p.tracks[:index], p.tracks[index+1:]...)
	p.revision++
	if p.shuffle {
		p.rebuildShuffleAfterRemove(index)
	}
	if p.current >= len(p.tracks) {
		p.current = len(p.tracks) - 1
	}
	if len(p.tracks) == 0 {
		p.current = -1
		p.shuffleOrder = nil
	} else if p.shuffle {
		p.shuffleIdx = p.findInShuffleOrder(p.current)
		p.shuffleStartIdx = p.shuffleIdx
	}
}

func (p *Playlist) Clear() {
	p.tracks = make([]Track, 0)
	p.revision++
	p.current = -1
	p.shuffleOrder = nil
	p.shuffleIdx = 0
	p.shuffleStartIdx = 0
}

func (p *Playlist) Current() (Track, bool) {
	if !p.isValidIndex(p.current) {
		return Track{}, false
	}
	return p.tracks[p.current], true
}

func (p *Playlist) CurrentIndex() int { return p.current }

func (p *Playlist) Tracks() []Track { return append([]Track(nil), p.tracks...) }

func (p *Playlist) At(index int) (Track, bool) {
	if !p.isValidIndex(index) {
		return Track{}, false
	}
	return p.tracks[index], true
}

func (p *Playlist) UpdateCurrentDuration(d time.Duration) {
	if !p.isValidIndex(p.current) {
		return
	}
	p.tracks[p.current].Duration = d
}

func (p *Playlist) Revision() uint64 { return p.revision }

func (p *Playlist) IsShuffling() bool           { return p.shuffle }
func (p *Playlist) Repeat() RepeatMode          { return p.repeat }
func (p *Playlist) SetRepeat(mode RepeatMode)   { p.repeat = mode }
func (p *Playlist) IsLast() bool                { return p.isLast() }
func (p *Playlist) IsValidIndex(index int) bool { return p.isValidIndex(index) }

func (p *Playlist) Next() (Track, bool) {
	if len(p.tracks) == 0 {
		return Track{}, false
	}
	if p.repeat == RepeatOne && p.isValidIndex(p.current) {
		return p.tracks[p.current], true
	}
	if p.shuffle {
		if len(p.shuffleOrder) == 0 {
			p.regenerateShuffle()
			if len(p.shuffleOrder) == 0 {
				return Track{}, false
			}
		}
		nextIdx := (p.shuffleIdx + 1) % len(p.shuffleOrder)
		if p.repeat == RepeatOff && nextIdx == p.shuffleStartIdx {
			return Track{}, false
		}
		p.shuffleIdx = nextIdx
		p.current = p.shuffleOrder[p.shuffleIdx]
		return p.tracks[p.current], true
	}
	if p.repeat == RepeatAll {
		p.current = (p.current + 1) % len(p.tracks)
		return p.tracks[p.current], true
	}
	if p.isLastSequential() {
		return Track{}, false
	}
	p.current++
	return p.tracks[p.current], true
}

func (p *Playlist) Previous() (Track, bool) {
	if len(p.tracks) == 0 || p.current < 0 {
		return Track{}, false
	}
	if p.shuffle {
		if len(p.shuffleOrder) == 0 {
			return Track{}, false
		}
		p.shuffleIdx--
		if p.shuffleIdx < 0 {
			p.shuffleIdx = len(p.shuffleOrder) - 1
		}
		p.current = p.shuffleOrder[p.shuffleIdx]
		return p.tracks[p.current], true
	}
	if p.current <= 0 {
		return Track{}, false
	}
	p.current--
	return p.tracks[p.current], true
}

func (p *Playlist) JumpTo(index int) bool {
	if !p.isValidIndex(index) {
		return false
	}
	p.current = index
	if p.shuffle {
		p.shuffleIdx = p.findInShuffleOrder(index)
		p.shuffleStartIdx = p.shuffleIdx
	}
	return true
}

func (p *Playlist) ToggleShuffle() {
	p.shuffle = !p.shuffle
	if p.shuffle && len(p.tracks) > 0 {
		p.regenerateShuffle()
	} else {
		p.shuffleOrder = nil
		p.shuffleIdx = 0
		p.shuffleStartIdx = 0
	}
}

func (p *Playlist) Length() int   { return len(p.tracks) }
func (p *Playlist) IsEmpty() bool { return len(p.tracks) == 0 }

func (p *Playlist) isLast() bool {
	if p.repeat != RepeatOff {
		return false
	}
	if p.shuffle {
		if len(p.shuffleOrder) == 0 {
			return true
		}
		return (p.shuffleIdx+1)%len(p.shuffleOrder) == p.shuffleStartIdx
	}
	return p.isLastSequential()
}

func (p *Playlist) isLastSequential() bool      { return p.current >= len(p.tracks)-1 }
func (p *Playlist) isValidIndex(index int) bool { return index >= 0 && index < len(p.tracks) }

func (p *Playlist) regenerateShuffle() {
	n := len(p.tracks)
	if n == 0 {
		return
	}
	p.shuffleOrder = make([]int, n)
	for i := range n {
		p.shuffleOrder[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		p.shuffleOrder[i], p.shuffleOrder[j] = p.shuffleOrder[j], p.shuffleOrder[i]
	}
	p.shuffleIdx = p.findInShuffleOrder(p.current)
	p.shuffleStartIdx = p.shuffleIdx
}

func (p *Playlist) rebuildShuffleAfterRemove(removedIndex int) {
	newOrder := make([]int, 0, len(p.shuffleOrder)-1)
	for _, idx := range p.shuffleOrder {
		if idx == removedIndex {
			continue
		}
		if idx > removedIndex {
			idx--
		}
		newOrder = append(newOrder, idx)
	}
	p.shuffleOrder = newOrder
}

func (p *Playlist) findInShuffleOrder(trackIndex int) int {
	for i, idx := range p.shuffleOrder {
		if idx == trackIndex {
			return i
		}
	}
	return 0
}
