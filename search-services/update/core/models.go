package core

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusIdle    ServiceStatus = "idle"
)

type EventType string

const (
	EventUpdate EventType = "update"
	EventReset  EventType = "reset"
)

type DBStats struct {
	WordsTotal    int64 `db:"words_total"`
	WordsUnique   int64 `db:"words_unique"`
	ComicsFetched int64 `db:"comics_fetched"`
}

type ServiceStats struct {
	DBStats
	ComicsTotal int64
}

type Comic struct {
	ID    int64    `db:"id"`
	URL   string   `db:"url"`
	Words []string `db:"words"`
}

type XKCDInfo struct {
	ID         int64  `json:"num"`
	URL        string `json:"img"`
	SafeTitle  string `json:"safe_title"`
	Title      string `json:"title"`
	Alt        string `json:"alt"`
	Transcript string `json:"transcript"`
}
