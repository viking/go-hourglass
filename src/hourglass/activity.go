package hourglass

import (
  "time"
  "strings"
)

type Activity struct {
  Id int64
  Name string
  Project string
  Tags []string
  Start time.Time
  End time.Time
}

func (a *Activity) TagList() string {
  return strings.Join(a.Tags, ", ")
}

func (a *Activity) SetTagList(tagList string) {
  a.Tags = strings.Split(tagList, ", ")
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
