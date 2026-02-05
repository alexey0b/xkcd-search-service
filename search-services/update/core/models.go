package core

type ServiceStatus string

const (
	StatusIdle    ServiceStatus = "idle"
	StatusRunning ServiceStatus = "running"
)

type EventType string

const (
	EventUpdate EventType = "update"
	EventReset  EventType = "reset"
)

// DBStats contains database statistics.
type DBStats struct {
	WordsTotal    int64 `db:"words_total"`
	WordsUnique   int64 `db:"words_unique"`
	ComicsFetched int64 `db:"comics_fetched"`
}

// ServiceStats contains database statistics and total comics count.
type ServiceStats struct {
	DBStats
	ComicsTotal int64
}

// Comic represents XKCD comic with normalized words.
type Comic struct {
	ID    int64    `db:"id"`
	URL   string   `db:"url"`
	Words []string `db:"words"`
}

// XKCDInfo contains comic data from XKCD API.
type XKCDInfo struct {
	ID         int64  `json:"num"`
	URL        string `json:"img"`
	SafeTitle  string `json:"safe_title"`
	Title      string `json:"title"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}
