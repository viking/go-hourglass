package hourglass

import (
  "testing"
  "time"
  "sort"
)

/* Activity sorting */
type activitySlice []*Activity

func (a activitySlice) Len() int {
  return len(a)
}

func (a activitySlice) Less(i, j int) bool {
  return a[i].Id < a[j].Id
}

func (a activitySlice) Swap(i, j int) {
  a[i], a[j] = a[j], a[i]
}

/* fakeDb */
type fakeDb struct {
  activityMap map[int64]*Activity
}

func (db *fakeDb) Valid() (bool, error) {
  return true, nil
}

func (db *fakeDb) Migrate() error {
  return nil
}

func (db *fakeDb) SaveActivity(a *Activity) error {
  if db.activityMap == nil {
    db.activityMap = make(map[int64]*Activity)
  }
  if a.Id == 0 {
    a.Id = int64(len(db.activityMap)) + 1
  }
  db.activityMap[a.Id] = a

  return nil
}

func (db *fakeDb) FindActivity(id int64) (*Activity, error) {
  activity, ok := db.activityMap[id]
  if !ok {
    return nil, ErrNotFound
  }
  return activity, nil
}

func (db *fakeDb) FindAllActivities() ([]*Activity, error) {
  activities := make(activitySlice, len(db.activityMap))
  i := 0
  for _, a := range(db.activityMap) {
    activities[i] = a
    i += 1
  }
  sort.Sort(activities)
  return activities, nil
}

func (db *fakeDb) FindRunningActivities() ([]*Activity, error) {
  var activities activitySlice
  for _, a := range db.activityMap {
    if a.IsRunning() {
      activities = append(activities, a)
    }
  }
  sort.Sort(activities)
  return activities, nil
}

func (db *fakeDb) FindActivitiesBetween(lower time.Time, upper time.Time) ([]*Activity, error) {
  var activities activitySlice
  for _, a := range db.activityMap {
    if a.Start.Equal(lower) || a.Start.After(lower) && a.Start.Before(upper) {
      activities = append(activities, a)
    }
  }
  sort.Sort(activities)
  return activities, nil
}

/* time helper */
func ago(d time.Duration) time.Time {
  return time.Now().UTC().Add(-d)
}

/* start command tests */
var startTests = []struct {
  name string
  project string
  tags []string
  output string
  err bool
}{
  {"", "", nil, "", true},
  {"foo", "", nil, "started activity 1", false},
  {"foo", "bar", nil, "started activity 1", false},
  {"foo", "bar", []string{"baz"}, "started activity 1", false},
}

func TestStartCommand_Run(t *testing.T) {
  for i, config := range startTests {
    c := StartCommand{}

    var args []string
    if config.name != "" {
      args = append(args, config.name)
      if config.project != "" {
        args = append(args, config.project)
        if len(config.tags) != 0 {
          args = append(args, config.tags...)
        }
      }
    }
    db := &fakeDb{}
    output, err := c.Run(db, args...)

    if output != config.output {
      t.Errorf("expected output to be '%s', but was '%s'", config.output, output)
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", i, err)
      }
      continue
    }
    if config.err {
      t.Errorf("test %d: expected error, got nil", i)
    }

    if len(db.activityMap) != 1 {
      t.Errorf("test %d: activity wasn't saved", i)
      continue
    }

    a := db.activityMap[1]
    if a.Name != config.name {
      t.Errorf("test %d: expected '%s', got '%s'", i, config.name, a.Name)
    }
    if a.Project != config.project {
      t.Errorf("test %d: expected '%s', got '%s'", i, config.project, a.Project)
    }

    ok := len(a.Tags) == len(config.tags)
    if ok {
      for i, tag := range config.tags {
        ok = tag == a.Tags[i]
        if !ok {
          break
        }
      }
    }
    if !ok {
      t.Errorf("test %d: expected %v, got %v", i, config.tags, a.Tags)
    }
  }
}

func TestStartCommand_Help(t *testing.T) {
  c := StartCommand{}
  if c.Help() == "" {
    t.Error("no help available")
  }
}

/* stop command tests */
var stopTests = []struct {
  startSince []time.Duration
  args []string
  runningAfter []bool
  output string
  err bool
}{
  /* Stop all when there are no args */
  {[]time.Duration{time.Hour, time.Hour}, nil, []bool{false, false}, "stopped activity 1\nstopped activity 2", false},
}

func TestStopCommand_Run(t *testing.T) {
  for i, config := range stopTests {
    c := StopCommand{}
    db := &fakeDb{}

    now := time.Now().UTC()
    for _, duration := range config.startSince {
      activity := &Activity{Name: "foo", Start: now.Add(-duration)}
      db.SaveActivity(activity)
    }

    output, err := c.Run(db, config.args...)
    if output != config.output {
      t.Errorf("expected output to be '%s', but was '%s'", config.output, output)
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", i, err)
      }
      continue
    }
    if config.err {
      t.Errorf("test %d: expected error, got nil", i)
    }

    for j, running := range config.runningAfter {
      activity, _ := db.FindActivity(int64(j + 1))
      if activity.IsRunning() != running {
        t.Errorf("expected %t, got %t", running, activity.IsRunning())
      }
      if !running && time.Since(activity.End) > time.Second {
        t.Errorf("activity's end time was wrong: %s", activity.End)
      }
    }
  }
}

func TestStopCommand_Help(t *testing.T) {
  c := StopCommand{}
  if c.Help() == "" {
    t.Error("no help available")
  }
}

var statusTests = []struct {
  activities []*Activity
  args []string
  output string
  err bool
}{
  {
    []*Activity{
      &Activity{Name: "foo", Tags: []string{"one", "two"}, Start: ago(time.Hour), End: ago(0)},
      &Activity{Name: "bar", Project: "baz", Start: ago(time.Hour)},
    },
    nil,
    "id\tname\tproject\ttags\tstate\tduration\n1\tfoo\t\tone, two\tstopped\t01h00m\n2\tbar\tbaz\t\trunning\t01h00m",
    false,
  },
}

func TestStatusCommand_Run(t *testing.T) {
  for i, config := range statusTests {
    c := StatusCommand{}
    db := &fakeDb{}

    for _, activity := range config.activities {
      db.SaveActivity(activity)
    }

    output, err := c.Run(db, config.args...)
    if output != config.output {
      t.Errorf("expected output to be '%s', but was '%s'", config.output, output)
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", i, err)
      }
      continue
    }
    if config.err {
      t.Errorf("test %d: expected error, got nil", i)
    }
  }
}
