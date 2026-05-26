package core

import "context"

type DB interface {
	Search(ctx context.Context, words []string) ([]Comic, error)
	IDs(ctx context.Context) ([]int, error)
	ListIDs(ctx context.Context, ids []int) ([]Comic, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type Scorer interface {
	ExtractScores(c Comic) map[string]int
	Score(c Comic, words []string) int
}

type SearchService interface {
	Search(ctx context.Context, query string, limit int32) ([]Comic, error)
	ISearch(ctx context.Context, query string, limit int32) ([]Comic, error)
	GetComic(ctx context.Context, id int) (Comic, error)
	DropIndex()
}

type Indexer interface {
	BuildIndex(ctx context.Context) error
	DropIndex()
}
