package core

import "context"

type Searcher interface {
	Search(ctx context.Context, phrase string, limit int) ([]Comic, error)
}

type MetaFetcher interface {
	FetchMeta(ctx context.Context, comicID int) (ComicMeta, error)
}

type StatsProvider interface {
	ComicsTotal(ctx context.Context) (int, error)
}

type AdminClient interface {
	Login(ctx context.Context, user, pass string) (string, error)
	Stats(ctx context.Context, token string) (Stats, error)
	Status(ctx context.Context, token string) (string, error)
	Update(ctx context.Context, token string) error
	Drop(ctx context.Context, token string) error
}
