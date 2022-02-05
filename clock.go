package pacemaker

import "time"

type RealClock struct{}

func (c RealClock) Now() time.Time {
	return time.Now()
}

func NewClock() *RealClock {
	return &RealClock{}
}

type TestClock struct {
	now time.Time
}

func (c TestClock) Now() time.Time {
	return c.now
}

func (c *TestClock) Forward(duration time.Duration) {
	c.now = c.now.Add(duration)
}

func NewMockClock(startAt time.Time) *TestClock {
	return &TestClock{now: startAt}
}
