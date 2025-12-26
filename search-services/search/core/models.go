package core

type EventType string

const (
	EventUpdate EventType = "update"
	EventReset  EventType = "reset"
)

type ComicInfo struct {
	Comic
	Words []string
}

type Comic struct {
	ID  int64  `db:"id"`
	URL string `db:"url"`
}
