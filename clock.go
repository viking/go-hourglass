package hourglass

import "time"

type Clock interface {
  Now() time.Time
  Local(time.Time) time.Time
  Since(time.Time) time.Duration
}

type DefaultClock struct {}

func (DefaultClock) Now() time.Time {
  return time.Now()
}

func (DefaultClock) Local(t time.Time) time.Time {
  return t.Local()
}

func (DefaultClock) Since(t time.Time) time.Duration {
  return time.Since(t)
}
