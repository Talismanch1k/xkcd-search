package score

import (
	"slices"

	"yadro.com/course/search/core"
)

const (
	ScoreTitle         = 5
	ScoreAlt           = 2
	ScoreTranscription = 1
)

type DefaultScorer struct{}

func (s DefaultScorer) ExtractScores(c core.Comic) map[string]int {
	wordScores := make(map[string]int)

	countWords := func(words []string, weigth int) {
		for _, w := range words {
			wordScores[w] += weigth
		}
	}

	countWords(c.Title, ScoreTitle)
	countWords(c.Alt, ScoreAlt)
	countWords(c.Transcript, ScoreTranscription)

	return wordScores
}

func (s DefaultScorer) Score(c core.Comic, words []string) int {
	score := 0
	for _, w := range words {
		if slices.Contains(c.Title, w) {
			score += ScoreTitle
		}
		if slices.Contains(c.Alt, w) {
			score += ScoreAlt
		}
		if slices.Contains(c.Transcript, w) {
			score += ScoreTranscription
		}
	}
	return score
}
