package words

import (
	"maps"
	"slices"
	"strings"
	"unicode"

	snowball "github.com/kljensen/snowball/english"
)

type WordsStemmer struct{}

func (WordsStemmer) Normalize(str string) []string {
	return Normalize(str)
}

// Normalize normalizes string, remove stop words and returns a slice of unique stemmed words
func Normalize(str string) []string {
	str = strings.ToLower(str)

	// maybe FieldsFuncSeq better
	words := strings.FieldsFunc(str, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})

	seen := make(map[string]struct{}, len(words))

	for _, w := range words {
		if snowball.IsStopWord(w) {
			continue
		}

		stm := snowball.Stem(w, false)

		if _, ok := seen[stm]; ok {
			continue
		}
		seen[stm] = struct{}{}

	}

	return slices.Collect(maps.Keys(seen))
}
