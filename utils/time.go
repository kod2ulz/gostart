package utils

import "time"

type timeUtils struct{}

var Time timeUtils

func (u timeUtils) StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
