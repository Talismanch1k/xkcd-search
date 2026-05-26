package score_test

import (
	"reflect"
	"testing"

	"yadro.com/course/search/adapters/score"
	"yadro.com/course/search/core"
)

func TestScore(t *testing.T) {
	t.Parallel()
	s := score.DefaultScorer{}

	comic := core.Comic{
		Title:      []string{"title", "titlealt", "titletrans", "titlealttrans"},
		Alt:        []string{"alt", "titlealt", "alttrans", "titlealttrans"},
		Transcript: []string{"transcript", "titletrans", "alttrans", "titlealttrans"},
	}

	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{
			name:  "word only in title",
			input: []string{"title"},
			want:  score.ScoreTitle,
		},
		{
			name:  "word only in alt",
			input: []string{"alt"},
			want:  score.ScoreAlt,
		},
		{
			name:  "word only in transcript",
			input: []string{"transcript"},
			want:  score.ScoreTranscription,
		},
		{
			name:  "word in title and alt",
			input: []string{"titlealt"},
			want:  score.ScoreTitle + score.ScoreAlt,
		},
		{
			name:  "word in title and transcript",
			input: []string{"titletrans"},
			want:  score.ScoreTitle + score.ScoreTranscription,
		},
		{
			name:  "word in alt and transcript",
			input: []string{"alttrans"},
			want:  score.ScoreAlt + score.ScoreTranscription,
		},
		{
			name:  "word in title, alt, and transcript",
			input: []string{"titlealttrans"},
			want:  score.ScoreTitle + score.ScoreAlt + score.ScoreTranscription,
		},
		{
			name:  "word not in comic",
			input: []string{"nonexistent"},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.Score(comic, tt.input)

			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtractScores(t *testing.T) {
	t.Parallel()
	s := score.DefaultScorer{}

	tests := []struct {
		name  string
		input core.Comic
		want  map[string]int
	}{
		{
			name: "comic with title, alt, and transcript",
			input: core.Comic{
				Title:      []string{"title", "titlealt", "titletrans", "titlealttrans"},
				Alt:        []string{"alt", "titlealt", "alttrans", "titlealttrans"},
				Transcript: []string{"transcript", "titletrans", "alttrans", "titlealttrans"},
			},
			want: map[string]int{
				"title":         score.ScoreTitle,
				"alt":           score.ScoreAlt,
				"transcript":    score.ScoreTranscription,
				"titlealt":      score.ScoreTitle + score.ScoreAlt,
				"titletrans":    score.ScoreTitle + score.ScoreTranscription,
				"alttrans":      score.ScoreAlt + score.ScoreTranscription,
				"titlealttrans": score.ScoreTitle + score.ScoreAlt + score.ScoreTranscription,
			},
		},
		{
			name: "comic with no alt",
			input: core.Comic{
				Title:      []string{"title", "titletrans"},
				Alt:        []string{},
				Transcript: []string{"transcript", "titletrans"},
			},
			want: map[string]int{
				"title":      score.ScoreTitle,
				"transcript": score.ScoreTranscription,
				"titletrans": score.ScoreTitle + score.ScoreTranscription,
			},
		},
		{
			name: "comic with no title",
			input: core.Comic{
				Title:      []string{},
				Alt:        []string{"alt", "alttrans"},
				Transcript: []string{"transcript", "alttrans"},
			},
			want: map[string]int{
				"alt":        score.ScoreAlt,
				"transcript": score.ScoreTranscription,
				"alttrans":   score.ScoreAlt + score.ScoreTranscription,
			},
		},
		{
			name: "comic with no transcript",
			input: core.Comic{
				Title:      []string{"title", "titlealt"},
				Alt:        []string{"alt", "titlealt"},
				Transcript: []string{},
			},
			want: map[string]int{
				"title":    score.ScoreTitle,
				"alt":      score.ScoreAlt,
				"titlealt": score.ScoreTitle + score.ScoreAlt,
			},
		},
		{
			name: "empty comic",
			input: core.Comic{
				Title:      []string{},
				Alt:        []string{},
				Transcript: []string{},
			},
			want: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := s.ExtractScores(tt.input)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
