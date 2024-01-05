package structs

import "time"

type ClickLusCode struct {
	LastError time.Time
	LastFetch time.Time
	LusCode   string
}

type ClickDesks struct {
	LastError time.Time
	LastFetch time.Time
	Status    string
}

type HarkNotification struct {
	Type             string
	StartramDaysLeft int
}
