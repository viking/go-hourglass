package hourglass

import (
  "testing"
  "time"
  "sort"
  "os"
  "os/exec"
  "syscall"
  "io/ioutil"
)

/* output diff */
func checkStringsEqual(expected, actual string) (ok bool, diff string, err error) {
  if expected == actual {
    ok = true
    return
  }

  var exp *os.File
  exp, err = ioutil.TempFile("", "expected")
  if err != nil {
    return
  }
  defer exp.Close()
  defer os.Remove(exp.Name())
  _, err = exp.Write([]byte(expected))
  if err != nil {
    return
  }
  exp.Sync()

  var act *os.File
  act, err = ioutil.TempFile("", "actual")
  if err != nil {
    return
  }
  defer act.Close()
  defer os.Remove(act.Name())
  _, err = act.Write([]byte(actual))
  if err != nil {
    return
  }
  act.Sync()

  // diff's exit status is 1 if the files differ, and Go returns an error
  // when the exit status is non-zero
  cmd := exec.Command("diff", "-u", exp.Name(), act.Name())
  var cmdOutput []byte
  cmdOutput, err = cmd.Output()
  if err != nil {
    var typeOk bool
    var exitErr *exec.ExitError
    exitErr, typeOk = err.(*exec.ExitError)
    if !typeOk {
      return
    }

    var status syscall.WaitStatus
    status, typeOk = exitErr.Sys().(syscall.WaitStatus)
    if !typeOk || status.ExitStatus() > 1 {
      return
    }
    err = nil
  }
  diff = string(cmdOutput)
  return
}

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
func (db *fakeDb) Version() (int, error) {
  return 1, nil
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
type fakeCmdClock struct {
  now time.Time
}
func (c fakeCmdClock) Now() time.Time {
  return c.now
}
func (c fakeCmdClock) Local(t time.Time) time.Time {
  return t.Local()
}
func (c fakeCmdClock) Since(t time.Time) time.Duration {
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
    c := fakeCmdClock{config.now}

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

    outputOk, diff, checkErr := checkStringsEqual(config.output, output)
    if !outputOk {
      if err == nil {
        t.Errorf("test %d: bad output:\n%s", i, diff)
      } else {
        t.Errorf("test %d: output didn't match, but couldn't create diff: %s", i, checkErr)
      }
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", i, err)
      } else if config.syntaxErr {
        _, ok := err.(SyntaxError)
        if !ok {
          t.Errorf("test %d: expected error type SyntaxError, got %T", i, err)
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

/* restart command tests */
var restartTests = []struct {
  now time.Time
  activity *Activity
  args []string
  output string
  err bool
  syntaxErr bool
}{
  {time.Now(), nil, nil, "", true, true},
  {time.Now(), nil, []string{"foo"}, "", true, true},
  {time.Now(), nil, []string{"1"}, "", true, false},
  {time.Now(), &Activity{Name: "foo"}, []string{"1"}, "restarted activity 1 (new id: 2)", false, false},
}

func TestRestartCommand_Run(t *testing.T) {
  for testNum, config := range restartTests {
    var err error

    cmd := RestartCommand{}
    db := &fakeDb{}
    c := fakeCmdClock{config.now}

    if config.activity != nil {
      err = db.SaveActivity(config.activity)
      if err != nil {
        t.Errorf("test %d: %s", testNum, err)
        continue
      }
      if len(db.activityMap) != 1 {
        t.Errorf("test %d: activity wasn't saved", testNum)
        continue
      }
    }

    var output string
    output, err = cmd.Run(c, db, config.args...)

    outputOk, diff, checkErr := checkStringsEqual(config.output, output)
    if !outputOk {
      if err == nil {
        t.Errorf("test %d: bad output:\n%s", testNum, diff)
      } else {
        t.Errorf("test %d: output didn't match, but couldn't create diff: %s", testNum, checkErr)
      }
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", testNum, err)
      } else if config.syntaxErr {
        _, ok := err.(SyntaxError)
        if !ok {
          t.Errorf("test %d: expected error type SyntaxError, got %T", testNum, err)
        }
      }
      continue
    }
    if config.err {
      t.Errorf("test %d: expected error, got nil", testNum)
    }

    if len(db.activityMap) != 2 {
      t.Errorf("test %d: activity wasn't saved", testNum)
      continue
    }

    a := db.activityMap[1]
    if a.Name != config.activity.Name {
      t.Errorf("test %d: expected '%s', got '%s'", testNum, config.activity.Name, a.Name)
    }
    if a.Project != config.activity.Project {
      t.Errorf("test %d: expected '%s', got '%s'", testNum, config.activity.Project, a.Project)
    }
    ok := len(a.Tags) == len(config.activity.Tags)
    if ok {
      for i, tag := range config.activity.Tags {
        ok = tag == a.Tags[i]
        if !ok {
          break
        }
      }
    }
    if !ok {
      t.Errorf("test %d: expected %v, got %v", testNum, config.activity.Tags, a.Tags)
    }
    if !a.Start.Equal(config.now) {
      t.Errorf("test %d: expected %v, got %v", testNum, config.now, a.Start)
    }
    if !a.End.IsZero() {
      t.Errorf("test %d: expected %v, got %v", testNum, time.Time{}, a.End)
    }
  }
}

func TestRestartCommand_Help(t *testing.T) {
  cmd := RestartCommand{}
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
    c := fakeCmdClock{config.now}

    now := c.Now()
    for _, duration := range config.startSince {
      activity := &Activity{Name: "foo", Start: now.Add(-duration)}
      db.SaveActivity(activity)
    }

    output, err := cmd.Run(c, db, config.args...)

    outputOk, diff, checkErr := checkStringsEqual(config.output, output)
    if !outputOk {
      if err == nil {
        t.Errorf("test %d: bad output:\n%s", i, diff)
      } else {
        t.Errorf("test %d: output didn't match, but couldn't create diff: %s", i, checkErr)
      }
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

var listTests = []struct {
  now time.Time
  activities []*Activity
  args []string
  output string
  err bool
}{
  /* test 0: listing activities */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "foo", Tags: []string{"one", "two"}, Start: when(2013, 4, 26, 14), End: when(2013, 4, 26, 15)},
      &Activity{Name: "bar", Project: "baz", Start: when(2013, 4, 26, 21)},
    },
    nil,
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 1\t| foo\t| \t| one, two\t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "| 2\t| bar\t| baz\t| \t| running\t| 21:00\t| \t| 01h00m\t|\n" +
    "baz: 01h00m, unsorted: 01h00m",
    false,
  },

  /* test 1: listing only today's activities */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "foo", Start: when(2013, 4, 25, 21), End: when(2013, 4, 25, 22)},
      &Activity{Name: "baz", Project: "proj", Start: when(2013, 4, 26, 14), End: when(2013, 4, 26, 15)},
      &Activity{Name: "bar", Start: when(2013, 4, 26, 21)},
    },
    nil,
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 2\t| baz\t| proj\t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "| 3\t| bar\t| \t| \t| running\t| 21:00\t| \t| 01h00m\t|\n" +
    "proj: 01h00m, unsorted: 01h00m",
    false,
  },

  /* test 2: output when there are no activities */
  {time.Now(), nil, nil, "there have been no activities today", false},

  /* test 3: week argument */
  {
    when(2013, 4, 27, 22),
    []*Activity{
      &Activity{Name: "sat", Start: when(2013, 4, 20, 14), End: when(2013, 4, 20, 15)},
      &Activity{Name: "sun", Start: when(2013, 4, 21, 14), End: when(2013, 4, 21, 15)},
      &Activity{Name: "mon", Start: when(2013, 4, 22, 14), End: when(2013, 4, 22, 15)},
      &Activity{Name: "wed", Start: when(2013, 4, 24, 14), End: when(2013, 4, 24, 15)},
      &Activity{Name: "thu", Start: when(2013, 4, 25, 14), End: when(2013, 4, 25, 15)},
      &Activity{Name: "fri", Start: when(2013, 4, 26, 14), End: when(2013, 4, 26, 15)},
      &Activity{Name: "sat", Start: when(2013, 4, 27, 21)},
    },
    []string{"week"},
    "=== Sunday (2013-04-21) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 2\t| sun\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "unsorted: 01h00m\n\n" +
    "=== Monday (2013-04-22) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 3\t| mon\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "unsorted: 01h00m\n\n" +
    "=== Wednesday (2013-04-24) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 4\t| wed\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "unsorted: 01h00m\n\n" +
    "=== Thursday (2013-04-25) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 5\t| thu\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "unsorted: 01h00m\n\n" +
    "=== Friday (2013-04-26) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 6\t| fri\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "unsorted: 01h00m\n\n" +
    "=== Saturday (2013-04-27) ===\n" +
    "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 7\t| sat\t| \t| \t| running\t| 21:00\t| \t| 01h00m\t|\n" +
    "unsorted: 01h00m",
    false,
  },

  /* test 4: all argument */
  {
    when(2013, 4, 26, 22),
    []*Activity{
      &Activity{Name: "baz", Start: when(2013, 4, 12, 14), End: when(2013, 4, 12, 15)},
      &Activity{Name: "foo", Start: when(2013, 4, 19, 14), End: when(2013, 4, 19, 15)},
      &Activity{Name: "bar", Start: when(2013, 4, 26, 21)},
    },
    []string{"all"},
    "| date\t| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|\n" +
    "| 2013-04-12\t| 1\t| baz\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "| 2013-04-19\t| 2\t| foo\t| \t| \t| stopped\t| 14:00\t| 15:00\t| 01h00m\t|\n" +
    "| 2013-04-26\t| 3\t| bar\t| \t| \t| running\t| 21:00\t| \t| 01h00m\t|",
    false,
  },

  /* all argument with no activities */
  {when(2013, 4, 26, 22), nil, []string{"all"}, "there aren't any activities", false},
}

func TestListCommand_Run(t *testing.T) {
  for i, config := range listTests {
    cmd := ListCommand{}
    db := &fakeDb{}
    c := fakeCmdClock{config.now}

    for _, activity := range config.activities {
      db.SaveActivity(activity)
    }

    output, err := cmd.Run(c, db, config.args...)

    outputOk, diff, checkErr := checkStringsEqual(config.output, output)
    if !outputOk {
      if err == nil {
        t.Errorf("test %d: bad output:\n%s", i, diff)
      } else {
        t.Errorf("test %d: output didn't match, but couldn't create diff: %s", i, checkErr)
      }
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

/* edit command tests */
var editTests = []struct {
  activityBefore *Activity
  args []string
  activityAfter *Activity
  output string
  err bool
}{
  /* test 0: no args */
  {nil, nil, nil, "", true},

  /* test 1: non-integer id */
  {nil, []string{"foo", "name", "bar"}, nil, "", true},

  /* test 2: id for missing activity */
  {nil, []string{"1", "name", "bar"}, nil, "", true},

  /* test 3: invalid field name */
  {
    &Activity{Name: "foo"},
    []string{"1", "blargh", "bar"},
    &Activity{Name: "foo"},
    "",
    true,
  },

  /* test 4: not enough arguments to name */
  {
    &Activity{Name: "foo"},
    []string{"1", "name"},
    &Activity{Name: "foo"},
    "",
    true,
  },

  /* test 5: change name */
  {
    &Activity{Name: "foo"},
    []string{"1", "name", "bar"},
    &Activity{Name: "bar"},
    "ok",
    false,
  },

  /* test 6: change name with extra args */
  {
    &Activity{Name: "foo"},
    []string{"1", "name", "bar", "baz"},
    &Activity{Name: "bar baz"},
    "ok",
    false,
  },

  /* test 7: change project */
  {
    &Activity{Project: "foo"},
    []string{"1", "project", "bar"},
    &Activity{Project: "bar"},
    "ok",
    false,
  },

  /* test 8: change project with extra args */
  {
    &Activity{Project: "foo"},
    []string{"1", "project", "bar", "baz"},
    &Activity{Project: "bar baz"},
    "ok",
    false,
  },

  /* test 9: remove project */
  {
    &Activity{Project: "foo"},
    []string{"1", "project"},
    &Activity{},
    "ok",
    false,
  },

  /* test 10: change tags */
  {
    &Activity{Tags: []string{"foo", "bar"}},
    []string{"1", "tags", "baz", "junk"},
    &Activity{Tags: []string{"baz", "junk"}},
    "ok",
    false,
  },

  /* test 11: remove tags */
  {
    &Activity{Tags: []string{"foo", "bar"}},
    []string{"1", "tags"},
    &Activity{Tags: []string{}},
    "ok",
    false,
  },

  /* test 12: change start to local time */
  {
    &Activity{Start: when(2013, 5, 13, 13)},
    []string{"1", "start", "2013-05-13 14:00"},
    &Activity{Start: when(2013, 5, 13, 14)},
    "ok",
    false,
  },

  /* test 13: change start to local time with multiple args */
  {
    &Activity{Start: when(2013, 5, 13, 13)},
    []string{"1", "start", "2013-05-13", "14:00"},
    &Activity{Start: when(2013, 5, 13, 14)},
    "ok",
    false,
  },

  /* test 14: not enough arguments for start */
  {
    &Activity{Start: when(2013, 5, 13, 13)},
    []string{"1", "start"},
    &Activity{Start: when(2013, 5, 13, 13)},
    "",
    true,
  },

  /* test 15: change start to time with zone */
  {
    &Activity{Start: when(2013, 5, 13, 13)},
    []string{"1", "start", "2013-05-13 15:00 -0400"},
    &Activity{Start: when(2013, 5, 13, 14)},
    "ok",
    false,
  },

  /* test 16: change end to local time */
  {
    &Activity{End: when(2013, 5, 13, 13)},
    []string{"1", "end", "2013-05-13 14:00"},
    &Activity{End: when(2013, 5, 13, 14)},
    "ok",
    false,
  },

  /* test 17: change end to local time with multiple args */
  {
    &Activity{End: when(2013, 5, 13, 13)},
    []string{"1", "end", "2013-05-13", "14:00"},
    &Activity{End: when(2013, 5, 13, 14)},
    "ok",
    false,
  },

  /* test 18: not enough arguments for end */
  {
    &Activity{End: when(2013, 5, 13, 13)},
    []string{"1", "end"},
    &Activity{End: when(2013, 5, 13, 13)},
    "",
    true,
  },

  /* test 19: change end to time with zone */
  {
    &Activity{End: when(2013, 5, 13, 13)},
    []string{"1", "end", "2013-05-13 15:00 -0400"},
    &Activity{End: when(2013, 5, 13, 14)},
    "ok",
    false,
  },
}

func TestEditCommand_Run(t *testing.T) {
  for testNum, config := range editTests {
    var err error

    cmd := EditCommand{}
    db := &fakeDb{}
    c := fakeCmdClock{time.Now()}

    if config.activityBefore != nil {
      err = db.SaveActivity(config.activityBefore)
      if err != nil {
        t.Errorf("test %d: %s", testNum, err)
        continue
      }
      config.activityAfter.Id = config.activityBefore.Id
    }

    var output string
    output, err = cmd.Run(c, db, config.args...)

    outputOk, diff, checkErr := checkStringsEqual(config.output, output)
    if !outputOk {
      if err == nil {
        t.Errorf("test %d: bad output:\n%s", testNum, diff)
      } else {
        t.Errorf("test %d: output didn't match, but couldn't create diff: %s", testNum, checkErr)
      }
    }

    if err != nil {
      if !config.err {
        t.Errorf("test %d: %s", testNum, err)
      }
      continue
    }
    if config.err {
      t.Errorf("test %d: expected error, got nil", testNum)
    }

    if config.activityBefore != nil {
      var foundActivity *Activity
      foundActivity, err = db.FindActivity(config.activityBefore.Id)
      if err != nil {
        t.Errorf("test %d: %s", testNum, err)
        continue
      }
      if !config.activityAfter.Equal(foundActivity) {
        t.Errorf("test %d: expected %v, got %v", testNum, config.activityAfter, foundActivity)
      }
    }
  }
}

func TestEditCommand_Help(t *testing.T) {
  cmd := EditCommand{}
  if cmd.Help() == "" {
    t.Error("no help available")
  }
}
