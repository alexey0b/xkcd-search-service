package core

type EventType string

const (
	EventUpdate EventType = "update"
	EventReset  EventType = "reset"
)

// ComicInfo extends Comic with normalized words.
type ComicInfo struct {
	Comic
	Words []string
}

// Comic represents a XKCD comic with ID and URL.
type Comic struct {
	ID  int64  `db:"id"`
	URL string `db:"url"`
}
