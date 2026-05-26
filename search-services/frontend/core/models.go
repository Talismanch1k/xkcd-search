package core

type Comic struct {
	ID    int
	URL   string
	Title string
	Alt   string
}

type ComicMeta struct {
	URL        string
	Title      string
	Alt        string
	Transcript string
}

type Stats struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
	ComicsTotal   int
}
