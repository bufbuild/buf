package bufsync

import "time"

type clock struct{}

func newClock() *clock {
	return &clock{}
}

func (*clock) Now() time.Time {
	return time.Now()
}
