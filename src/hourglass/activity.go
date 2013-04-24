package hourglass

import "time"

type Activity struct {
  Name string
  Project string
  Tags []string
  Start time.Time
  End time.Time
}

func (a *Activity) Duration() time.Duration {
  if a.IsRunning() {
    return time.Since(a.Start)
  }
  return a.End.Sub(a.Start)
}

func (a *Activity) IsRunning() bool {
  return a.End.IsZero()
}
