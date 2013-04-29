package hourglass

import (
  "time"
  "strings"
  "fmt"
)

type Activity struct {
  Id int64
  Name string
  Project string
  Tags []string
  Start time.Time
  End time.Time
}

type Duration time.Duration

func (d Duration) String() string {
  hours := int64(d) / int64(time.Hour)
  minutes := int64(d) % int64(time.Hour) / int64(time.Minute)
  return fmt.Sprintf("%02dh%02dm", hours, minutes)
}

func (a *Activity) TagList() string {
  return strings.Join(a.Tags, ", ")
}

func (a *Activity) SetTagList(tagList string) {
  if tagList == "" {
    a.Tags = nil
  } else {
    a.Tags = strings.Split(tagList, ", ")
  }
}

func (a *Activity) Duration() Duration {
  if a.IsRunning() {
    return Duration(time.Since(a.Start))
  }
  return Duration(a.End.Sub(a.Start))
}

func (a *Activity) IsRunning() bool {
  return a.End.IsZero()
}

func (a *Activity) Equal(b *Activity) bool {
  if a.Id != b.Id {
    return false
  }
  if a.Name != b.Name {
    return false
  }
  if a.Project != b.Project {
    return false
  }
  if len(a.Tags) != len(b.Tags) {
    return false
  }
  for i, tag := range a.Tags {
    if b.Tags[i] != tag {
      return false
    }
  }
  if !a.Start.Equal(b.Start) {
    return false
  }
  if !a.End.Equal(b.End) {
    return false
  }
  return true
}

func (a *Activity) Status() string {
  if a.IsRunning() {
    return "running"
  }
  return "stopped"
}

func (a *Activity) Clone() *Activity {
  b := &Activity{a.Id, a.Name, a.Project, nil, a.Start, a.End}
  b.Tags = make([]string, len(a.Tags))
  copy(b.Tags, a.Tags)
  return b
}
