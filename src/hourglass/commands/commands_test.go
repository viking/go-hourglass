package commands

import (
  "testing"
  "time"
  "sort"
  . "hourglass/activity"
  . "hourglass/database"
)

/* activity sorting */
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

/* fake database */
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

/* fake clock */
type fakeClock struct {
  now time.Time
}
func (c fakeClock) Now() time.Time {
  return c.now
}
func (c fakeClock) Local(t time.Time) time.Time {
  return t.Local()
}
func (c fakeClock) Since(t time.Time) time.Duration {
  return c.now.Sub(t)
}

/* time helpers */
func when(year, month, day, hour int) time.Time {
  return time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.Local)
}

/* start command tests */
var startTests = []struct {
  now time.Time
  name string
  project string
  tags []string
  output string
  err bool
  syntaxErr bool
}{
  {time.Now(), "", "", nil, "", true, true},
  {time.Now(), "foo", "", nil, "started activity 1", false, false},
  {time.Now(), "foo", "bar", nil, "started activity 1", false, false},
  {time.Now(), "foo", "bar", []string{"baz"}, "started activity 1", false, false},
}

func TestStartCommand_Run(t *testing.T) {
  for i, config := range startTests {
    cmd := StartCommand{}
    db := &fakeDb{}
    c := fakeClock{config.now}

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
    output, err := cmd.Run(c, db, args...)

    if output != config.output {
      t.Errorf("expected output to be '%s', but was '%s'", config.output, output)
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", i, err)
      } else if config.syntaxErr {
        _, ok := err.(SyntaxErr)
        if !ok {
          t.Errorf("test %d: expected error type SyntaxErr, got %T", i, err)
        }
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
  cmd := StartCommand{}
  if cmd.Help() == "" {
    t.Error("no help available")
  }
}

/* stop command tests */
var stopTests = []struct {
  now time.Time
  startSince []time.Duration
  args []string
  runningAfter []bool
  output string
  err bool
}{
  /* Stop all when there are no args */
  {time.Now(), []time.Duration{time.Hour, time.Hour}, nil, []bool{false, false}, "stopped activity 1\nstopped activity 2", false},
}

func TestStopCommand_Run(t *testing.T) {
  for i, config := range stopTests {
    cmd := StopCommand{}
    db := &fakeDb{}
    c := fakeClock{config.now}

    now := c.Now()
    for _, duration := range config.startSince {
      activity := &Activity{Name: "foo", Start: now.Add(-duration)}
      db.SaveActivity(activity)
    }

    output, err := cmd.Run(c, db, config.args...)
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
  cmd := StopCommand{}
  if cmd.Help() == "" {
    t.Error("no help available")
  }
}

var statusTests = []struct {
  now time.Time
  activities []*Activity
  args []string
  output string
  err bool
}{
  /* listing activities */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "foo", Tags: []string{"one", "two"}, Start: when(2013, 4, 26, 14), End: when(2013, 4, 26, 15)},
      &Activity{Name: "bar", Project: "baz", Start: when(2013, 4, 26, 21)},
    },
    nil,
    "| id\t| name\t| project\t| tags\t| state\t| duration\n" +
      "| 1\t| foo\t| \t| one, two\t| stopped\t| 01h00m\n" +
      "| 2\t| bar\t| baz\t| \t| running\t| 01h00m",
    false,
  },

  /* listing only today's activities */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "foo", Start: when(2013, 4, 25, 21), End: when(2013, 4, 25, 22)},
      &Activity{Name: "bar", Start: when(2013, 4, 26, 21)},
    },
    nil,
    "| id\t| name\t| project\t| tags\t| state\t| duration\n" +
      "| 2\t| bar\t| \t| \t| running\t| 01h00m",
    false,
  },

  /* output when there are no activities */
  {time.Now(), nil, nil, "there have been no activities today", false},

  /* all argument */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "baz", Start: when(2013, 4, 12, 14), End: when(2013, 4, 12, 15)},
      &Activity{Name: "foo", Start: when(2013, 4, 19, 14), End: when(2013, 4, 19, 15)},
      &Activity{Name: "bar", Start: when(2013, 4, 26, 21)},
    },
    []string{"all"},
    "| date\t| id\t| name\t| project\t| tags\t| state\t| duration\n" +
      "| 2013-04-12\t| 1\t| baz\t| \t| \t| stopped\t| 01h00m\n" +
      "| 2013-04-19\t| 2\t| foo\t| \t| \t| stopped\t| 01h00m\n" +
      "| 2013-04-26\t| 3\t| bar\t| \t| \t| running\t| 01h00m",
    false,
  },

  /* all argument with no activities */
  {when(2013, 4, 26, 22), nil, []string{"all"}, "there aren't any activities", false},
}

func TestStatusCommand_Run(t *testing.T) {
  for i, config := range statusTests {
    cmd := StatusCommand{}
    db := &fakeDb{}
    c := fakeClock{config.now}

    for _, activity := range config.activities {
      db.SaveActivity(activity)
    }

    output, err := cmd.Run(c, db, config.args...)
    if output != config.output {
      t.Errorf("test %d: expected output to be '%s', but was '%s'", i, config.output, output)
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
