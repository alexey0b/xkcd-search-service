package core

type (
	PingStatus   string
	UpdateStatus string

	ContextKey string
)

const JwtTokenContextKey ContextKey = "jwt_token"

type PingResponse struct {
	Replies map[string]PingStatus `json:"replies"`
}

type UpdateStats struct {
	WordsTotal    int64 `json:"words_total"`
	WordsUnique   int64 `json:"words_unique"`
	ComicsFetched int64 `json:"comics_fetched"`
	ComicsTotal   int64 `json:"comics_total"`
}

type Comic struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type SearchResult struct {
	Comics []Comic `json:"comics"`
	Total  int64   `json:"total"`
}
