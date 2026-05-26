package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(ctx context.Context, query string, limit int32) (SearchResult, error)
	ISearch(ctx context.Context, query string, limit int32) (SearchResult, error)
	GetComic(ctx context.Context, id int) (ComicInfo, error)
	Drop(ctx context.Context) error
}
