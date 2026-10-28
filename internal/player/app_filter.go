package player

import (
	"sort"
	"strings"
	"unicode/utf8"
)

type searchCandidate struct {
	normalized string
	runeLength int
}

type filterMatch struct {
	index int
	score int
}

// rebuildFilteredIndices calcula los resultados y sugerencias conservando el ranking actual.
func (m *AppModel) rebuildFilteredIndices() {
	trackCount := m.playlist.Length()
	m.filteredIndices = make([]int, 0, trackCount)
	m.suggestionIndices = make([]int, 0, max(0, min(m.suggestionLimit, trackCount)))

	query := strings.ToLower(strings.TrimSpace(m.filterQuery))
	if query == "" {
		for i := 0; i < trackCount; i++ {
			m.filteredIndices = append(m.filteredIndices, i)
		}
		m.copySuggestions()
		return
	}

	m.ensureSearchCandidates()
	queryRunes := []rune(query)
	matches := make([]filterMatch, 0, trackCount)
	for i, candidate := range m.searchCandidates {
		if position := strings.Index(candidate.normalized, query); position >= 0 {
			score := 100 + max(0, 30-position*2)
			if strings.HasPrefix(candidate.normalized, query) {
				score += 50
			}
			score += max(0, 50-candidate.runeLength)
			matches = append(matches, filterMatch{index: i, score: score})
			continue
		}

		if isRuneSubsequence(queryRunes, candidate.normalized) {
			score := max(1, 30-candidate.runeLength/10)
			matches = append(matches, filterMatch{index: i, score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	for _, match := range matches {
		m.filteredIndices = append(m.filteredIndices, match.index)
	}
	m.copySuggestions()
}

func (m *AppModel) ensureSearchCandidates() {
	if m.searchRevision == m.playlist.revision && len(m.searchCandidates) == m.playlist.Length() {
		return
	}

	m.searchCandidates = make([]searchCandidate, 0, m.playlist.Length())
	for _, track := range m.playlist.tracks {
		normalized := strings.ToLower(track.DisplayName())
		m.searchCandidates = append(m.searchCandidates, searchCandidate{
			normalized: normalized,
			runeLength: utf8.RuneCountInString(normalized),
		})
	}
	m.searchRevision = m.playlist.revision
}

func (m *AppModel) copySuggestions() {
	limit := max(0, min(m.suggestionLimit, len(m.filteredIndices)))
	m.suggestionIndices = append(m.suggestionIndices, m.filteredIndices[:limit]...)
}

func isRuneSubsequence(query []rune, candidate string) bool {
	if len(query) == 0 {
		return true
	}

	queryIndex := 0
	for _, candidateRune := range candidate {
		if query[queryIndex] == candidateRune {
			queryIndex++
			if queryIndex == len(query) {
				return true
			}
		}
	}
	return false
}
