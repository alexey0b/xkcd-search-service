package core

type UpdateStatus string

type ContextKey string

// JwtTokenContextKey is the context key for storing JWT token.
const JwtTokenContextKey ContextKey = "jwt_token"

type HealthStatus string

// HealthResponse contains health status of all registered services.
type HealthResponse struct {
	Replies map[string]HealthStatus `json:"replies"`
}

// UpdateStats contains database statistics.
type UpdateStats struct {
	WordsTotal    int64 `json:"words_total"`
	WordsUnique   int64 `json:"words_unique"`
	ComicsFetched int64 `json:"comics_fetched"`
	ComicsTotal   int64 `json:"comics_total"`
}

// Comic represents a single XKCD comic with ID and URL.
type Comic struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// SearchResult contains search results with matching comics.
type SearchResult struct {
	Comics []Comic `json:"comics"`
	Total  int64   `json:"total"`
}
