package clock

import "time"

type Clock interface {
  Now() time.Time
  Local(time.Time) time.Time
  Since(time.Time) time.Duration
}

type RealClock struct {}

func (RealClock) Now() time.Time {
  return time.Now()
}

func (RealClock) Local(t time.Time) time.Time {
  return t.Local()
}

func (RealClock) Since(t time.Time) time.Duration {
  return time.Since(t)
}
