package core

type (
	PingStatus   string
	UpdateStatus string
)

const (
	StatusPingOK          PingStatus = "ok"
	StatusPingUnavailable PingStatus = "unavailable"

	StatusUpdateUnknown UpdateStatus = "unknown"
	StatusUpdateIdle    UpdateStatus = "idle"
	StatusUpdateRunning UpdateStatus = "running"
)

type PingResponse struct {
	Replies map[string]PingStatus `json:"replies"`
}

type UpdateStatusResponse struct {
	Status UpdateStatus `json:"status"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Comic struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
}

type UpdateStats struct {
	WordsTotal    int64 `json:"words_total"`
	WordsUnique   int64 `json:"words_unique"`
	ComicsFetched int64 `json:"comics_fetched"`
	ComicsTotal   int64 `json:"comics_total"`
}

type SearchResult struct {
	Comics []Comic `json:"comics"`
	Total  int64   `json:"total"`
}
