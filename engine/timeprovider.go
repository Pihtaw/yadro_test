package engine

import "time"

type TimeProvider interface {
	Parse(layout, value string) (time.Time, error)
	Now() time.Time
}

type RealTimeProvider struct{}

func (RealTimeProvider) Parse(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

func (RealTimeProvider) Now() time.Time {
	return time.Now()
}
