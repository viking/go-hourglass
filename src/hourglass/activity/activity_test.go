package activity

import (
  "testing"
  "time"
)

func TestActivity_Duration(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar"}
  activity.Start = time.Date(2013, time.April, 24, 13, 0, 0, 0, time.UTC)

  duration := time.Duration(time.Hour)
  activity.End = activity.Start.Add(duration)
  result := activity.Duration()
  if Duration(duration) != result {
    t.Error("expected", duration, "got", result)
  }
}

func TestActivity_Duration_WithNoEnd(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar"}
  activity.Start = time.Now().Add(time.Duration(-time.Hour))

  duration := time.Since(activity.Start)
  result := time.Duration(activity.Duration())
  if result - duration > time.Microsecond {
    t.Error("expected", duration, "got", result)
  }
}

func TestActivity_IsRunning(t *testing.T) {
  activity := Activity{Name: "foo", Project: "bar", Start: time.Now()}
  if !activity.IsRunning() {
    t.Error("expected activity to be running")
  }
}

func TestActivity_TagList(t *testing.T) {
  activity := Activity{Tags: []string{"foo", "bar", "baz"}}
  expected := "foo, bar, baz"
  actual := activity.TagList()
  if expected != actual {
    t.Error("expected", expected, "got", actual)
  }
}

func TestActivity_SetTagList(t *testing.T) {
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

func TestActivity_SetTagList_WithEmptyString(t *testing.T) {
  activity := Activity{}
  activity.SetTagList("")

  if len(activity.Tags) != 0 {
    t.Error("expected tags array to be empty")
  }
}

func TestActivity_Equal(t *testing.T) {
  end := time.Now()
  start := end.Add(-time.Duration(time.Hour))
  activity_1 := &Activity{1, "foo", "bar", []string{"baz"}, start, end}
  activity_2 := &Activity{1, "foo", "bar", []string{"baz"}, start, end}
  if !activity_1.Equal(activity_2) {
    t.Error("expected activities to be equal")
  }
}

func TestActivity_Status(t *testing.T) {
  activity := &Activity{1, "foo", "bar", []string{}, time.Now(), time.Time{}}
  if activity.Status() != "running" {
    t.Errorf("expected 'running', got '%s'", activity.Status())
  }

  activity.End = time.Now()
  if activity.Status() != "stopped" {
    t.Errorf("expected 'stopped', got '%s'", activity.Status())
  }
}

func TestActivity_Clone(t *testing.T) {
  end := time.Now()
  start := end.Add(-time.Duration(time.Hour))
  activity_1 := &Activity{1, "foo", "bar", []string{"baz"}, start, end}
  activity_2 := activity_1.Clone()

  activity_2.Name = "qux"
  if activity_1.Name == activity_2.Name {
    t.Error("activity wasn't cloned")
  }
  activity_2.Tags[0] = "junk"
  if activity_1.Tags[0] == activity_2.Tags[0] {
    t.Error("activity tags weren't cloned")
  }
}

var durationTests = []struct {
  num int64
  output string
}{
  {int64(time.Hour), "01h00m"},
  {int64((4 * time.Hour) + (30 * time.Minute)), "04h30m"},
}

func TestDuration_String(t *testing.T) {
  for _, config := range durationTests {
    duration := Duration(config.num)
    if duration.String() != config.output {
      t.Errorf("expected '%s', got '%s'", config.output, duration.String())
    }
  }
}
