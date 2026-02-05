package core

// HealthStatus represents the health state of a service.
type (
	HealthStatus string
	UpdateStatus string
)

const (
	HealthOK          HealthStatus = "ok"
	HealthUnavailable HealthStatus = "unavailable"

	UpdateUnknown UpdateStatus = "unknown"
	UpdateIdle    UpdateStatus = "idle"
	UpdateRunning UpdateStatus = "running"
)

// HealthResponse contains health status of all registered services.
type HealthResponse struct {
	Replies map[string]HealthStatus `json:"replies"`
}

// UpdateStatusResponse wraps the current update process status.
type UpdateStatusResponse struct {
	Status UpdateStatus `json:"status"`
}

// LoginRequest contains admin authentication credentials.
type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Comic represents a XKCD comic with ID and URL.
type Comic struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

// UpdateStats contains database statistics.
type UpdateStats struct {
	WordsTotal    int64 `json:"words_total"`
	WordsUnique   int64 `json:"words_unique"`
	ComicsFetched int64 `json:"comics_fetched"`
	ComicsTotal   int64 `json:"comics_total"`
}

// SearchResult contains search results with matching comics.
type SearchResult struct {
	Comics []Comic `json:"comics"`
	Total  int64   `json:"total"`
}
