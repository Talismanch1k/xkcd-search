package words_test

import (
	"slices"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"yadro.com/course/words/words"
)

func TestWordsStemmer_Normalize(t *testing.T) {
	s := words.WordsStemmer{}
	got := s.Normalize("follower brings bunch of questions")
	if len(got) == 0 {
		t.Error("expected non empty result")
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "example",
			input: "follower brings bunch of questions",
			want:  []string{"follow", "bring", "bunch", "question"},
		},
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only stop words",
			input: "the a of will is are",
			want:  []string{},
		},
		{
			name:  "mixed case",
			input: "Running JUMPING Swimming",
			want:  []string{"run", "jump", "swim"},
		},
		{
			name:  "pronouns filtered",
			input: "he brings her flowers",
			want:  []string{"bring", "flower"},
		},
		{
			name:  "different CaSeS",
			input: "hE brINgs HeR FlowER AND are gOOd",
			want:  []string{"bring", "flower", "good"},
		},
		{
			name:  "punctuation",
			input: "can. you be my friend? Are, you Lion?",
			want:  []string{"friend", "lion"},
		},
		{
			name:  "duplicate",
			input: "super super super lion",
			want:  []string{"super", "lion"},
		},
		{
			name:  "numbers",
			input: "super 42 super 52 dog 67 lion 812",
			want:  []string{"super", "dog", "lion", "42", "52", "67", "812"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := words.Normalize(tt.input)

			sort.Strings(got)
			sort.Strings(tt.want)

			if !slices.Equal(got, tt.want) {
				t.Errorf("Normalize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func FuzzNormalize(f *testing.F) {
	f.Add("hello world")
	f.Add("")
	f.Add("the a of")
	f.Add("Running JUMPING")
	f.Add("hello hello hello")
	f.Add("привет мир")
	f.Add("!@#$%^&*()")
	f.Add("0123456789")

	f.Fuzz(func(t *testing.T, input string) {
		res := words.Normalize(input)

		// not nil
		if res == nil && len(res) != 0 {
			t.Errorf("Normalize(%q) returned nil", input)
		}

		// no duplicates
		seen := make(map[string]struct{}, len(res))
		for _, w := range res {
			if _, ok := seen[w]; ok {
				t.Errorf("Normalize(%q) has duplicate %q", input, w)
			}
		}

		// all words in lower
		for _, w := range res {
			if w != strings.ToLower(w) {
				t.Errorf("Normalize(%q) returned uppercase word %q", input, w)
			}
		}

		// valid utf8
		for _, w := range res {
			if !utf8.ValidString(w) {
				t.Errorf("Normalize(%q) return not valid utf8 %q", input, w)
			}
		}
	})
}
