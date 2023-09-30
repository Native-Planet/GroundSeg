package structs

import "time"

type ClickLusCode struct {
	LastError time.Time
	LastFetch time.Time
	LusCode   string
}
