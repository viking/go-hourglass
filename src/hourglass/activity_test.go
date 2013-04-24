package hourglass

import (
  "testing"
  "time"
)

func TestDuration(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar"}
  activity.Start = time.Date(2013, time.April, 24, 13, 0, 0, 0, time.UTC)

  duration := time.Duration(time.Hour)
  activity.End = activity.Start.Add(duration)
  result := activity.Duration()
  if duration != result {
    t.Error("expected", duration, "got", result)
  }
}

func TestDurationWithNoEnd(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar"}
  activity.Start = time.Now().Add(time.Duration(-time.Hour))

  duration := time.Since(activity.Start)
  result := activity.Duration()
  if result - duration > time.Microsecond {
    t.Error("expected", duration, "got", result)
  }
}

func TestIsRunning(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar", Start: time.Now()}
  if !activity.IsRunning() {
    t.Error("expected activity to be running")
  }
}

func TestTagList(t *testing.T) {
  activity := Activity{Tags: []string{"foo", "bar", "baz"}}
  expected := "foo, bar, baz"
  actual := activity.TagList()
  if expected != actual {
    t.Error("expected", expected, "got", actual)
  }
}

func TestSetTagList(t *testing.T) {
  activity := Activity{}
  activity.SetTagList("foo, bar, baz")

  expected := []string{"foo", "bar", "baz"}
  fail := len(activity.Tags) != len(expected)
  if !fail {
    for i := 0; i < len(expected); i++ {
      if expected[i] != activity.Tags[i] {
        fail = true
        break
      }
    }
  }
  if fail {
    t.Error("expected", expected, "got", activity.Tags)
  }
}
