package hourglass

import (
  "testing"
  "time"
  "fmt"
  "strings"
)

func TestStartCommand_Run_WithMissingName(t *testing.T) {
  f := func (db *Database) {
    c := StartCommand{}
    _, cmdErr := c.Run(db)
    if cmdErr == nil {
      t.Error("expected command error, but there wasn't one")
    }
  }
  DbTestRun(f, t)
}

func TestStartCommand_Run_WithName(t *testing.T) {
  f := func (db *Database) {
    c := StartCommand{}
    cmdOutput, cmdErr := c.Run(db, "foo")
    if cmdErr != nil {
      t.Error(cmdErr)
      return
    }

    activities, findErr := db.FindAllActivities()
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected 1 activity, got", len(activities))
    }
    if activities[0].Name != "foo" {
      t.Error("expected name to be foo, but was", activities[0].Name)
    }
    if activities[0].Project != "" {
      t.Error("expected project to be empty, but was", activities[0].Project)
    }
    if len(activities[0].Tags) != 0 {
      t.Error("expected tags to be empty, but was", activities[0].Tags)
    }

    duration := time.Since(activities[0].Start)
    if duration > time.Second {
      t.Error("expected start time to be", time.Now(), "but was",
        activities[0].Start.Local())
    }

    if !activities[0].End.IsZero() {
      t.Error("expected end time to be zero, but was", activities[0].End)
    }

    expOutput := fmt.Sprintf("started activity %d", activities[0].Id)
    if cmdOutput != expOutput {
      t.Errorf("expected output to be '%s' but was '%s'", expOutput, cmdOutput)
    }
  }
  DbTestRun(f, t)
}

func TestStartCommand_Run_WithNameAndProject(t *testing.T) {
  f := func (db *Database) {
    c := StartCommand{}
    cmdOutput, cmdErr := c.Run(db, "foo", "bar")
    if cmdErr != nil {
      t.Error(cmdErr)
      return
    }

    activities, findErr := db.FindAllActivities()
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected 1 activity, got", len(activities))
    }
    if activities[0].Name != "foo" {
      t.Error("expected name to be foo, but was", activities[0].Name)
    }
    if activities[0].Project != "bar" {
      t.Error("expected project to be bar, but was", activities[0].Project)
    }
    if len(activities[0].Tags) != 0 {
      t.Error("expected tags to be empty, but was", activities[0].Tags)
    }

    duration := time.Since(activities[0].Start)
    if duration > time.Second {
      t.Error("expected start time to be", time.Now(), "but was",
        activities[0].Start.Local())
    }

    if !activities[0].End.IsZero() {
      t.Error("expected end time to be zero, but was", activities[0].End)
    }

    expOutput := fmt.Sprintf("started activity %d", activities[0].Id)
    if cmdOutput != expOutput {
      t.Errorf("expected output to be '%s' but was '%s'", expOutput, cmdOutput)
    }
  }
  DbTestRun(f, t)
}

func TestStartCommand_Run_WithAllAttribs(t *testing.T) {
  f := func (db *Database) {
    c := StartCommand{}
    cmdOutput, cmdErr := c.Run(db, "foo", "bar", "baz", "qux")
    if cmdErr != nil {
      t.Error(cmdErr)
      return
    }

    activities, findErr := db.FindAllActivities()
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected 1 activity, got", len(activities))
    }
    if activities[0].Name != "foo" {
      t.Error("expected name to be foo, but was", activities[0].Name)
    }
    if activities[0].Project != "bar" {
      t.Error("expected project to be bar, but was", activities[0].Project)
    }
    if len(activities[0].Tags) != 2 || activities[0].Tags[0] != "baz" || activities[0].Tags[1] != "qux" {
      t.Error("expected tags to be baz, but was", activities[0].Tags)
    }

    duration := time.Since(activities[0].Start)
    if duration > time.Second {
      t.Error("expected start time to be", time.Now(), "but was",
        activities[0].Start.Local())
    }

    if !activities[0].End.IsZero() {
      t.Error("expected end time to be zero, but was", activities[0].End)
    }

    expOutput := fmt.Sprintf("started activity %d", activities[0].Id)
    if cmdOutput != expOutput {
      t.Errorf("expected output to be '%s' but was '%s'", expOutput, cmdOutput)
    }
  }
  DbTestRun(f, t)
}

func TestStartCommand_Help(t *testing.T) {
  c := StartCommand{}
  if c.Help() == "" {
    t.Error("no help available")
  }
}

func TestStopCommand_Run_WithNoArgs(t *testing.T) {
  f := func (db *Database) {
    start := time.Now().Add(-time.Hour)
    activity_1 := &Activity{Name: "foo", Start: start}
    activity_2 := &Activity{Name: "bar", Start: start}

    var saveErr error
    saveErr = db.SaveActivity(activity_1)
    if saveErr != nil {
      t.Error(saveErr)
    }
    saveErr = db.SaveActivity(activity_2)
    if saveErr != nil {
      t.Error(saveErr)
    }

    c := StopCommand{}
    cmdOutput, cmdErr := c.Run(db)
    if cmdErr != nil {
      t.Error(cmdErr)
    }

    expected := time.Now()

    var foundActivity_1, foundActivity_2 *Activity
    var findErr error
    foundActivity_1, findErr = db.FindActivity(activity_1.Id)
    if findErr != nil {
      t.Error(findErr)
    } else {
      duration := expected.Sub(foundActivity_1.End)
      if duration > time.Second {
        t.Error("expected activity 1's end time to be", expected, "but was",
          foundActivity_1.End)
      }
    }

    foundActivity_2, findErr = db.FindActivity(activity_2.Id)
    if findErr != nil {
      t.Error(findErr)
    } else {
      duration := expected.Sub(foundActivity_2.End)
      if duration > time.Second {
        t.Error("expected activity 2's end time to be", expected, "but was",
          foundActivity_2.End)
      }
    }

    expOutput := fmt.Sprintf("stopped activity %d\nstopped activity %d",
      foundActivity_1.Id, foundActivity_2.Id)
    if cmdOutput != expOutput {
      t.Errorf("expected output to be '%s' but was '%s'", expOutput, cmdOutput)
    }
  }
  DbTestRun(f, t)
}

func TestStopCommand_Help(t *testing.T) {
  c := StopCommand{}
  if c.Help() == "" {
    t.Error("no help available")
  }
}

func TestStatusCommand_Run_WithNoArgs(t *testing.T) {
  f := func (db *Database) {
    now := time.Now().UTC()
    start := now.Add(-time.Hour)
    activity_1 := &Activity{Name: "foo", Start: start, End: now}
    activity_2 := &Activity{Name: "bar", Project: "baz", Start: start}

    var saveErr error
    saveErr = db.SaveActivity(activity_1)
    if saveErr != nil {
      t.Error(saveErr)
    }
    saveErr = db.SaveActivity(activity_2)
    if saveErr != nil {
      t.Error(saveErr)
    }

    c := StatusCommand{}
    cmdOutput, cmdErr := c.Run(db)
    if cmdErr != nil {
      t.Error(cmdErr)
    }

    expLine1 := fmt.Sprint("id\tname\tproject\tstate\tduration")
    if !strings.Contains(cmdOutput, expLine1) {
      t.Errorf("expected output to contain '%s'", expLine1)
    }

    expLine2 := fmt.Sprintf("%d\t%s\t%s\t%s\t%s", activity_1.Id,
      activity_1.Name, activity_1.Project, activity_1.Status(),
      activity_1.Duration().String())
    if !strings.Contains(cmdOutput, expLine2) {
      t.Errorf("expected output to contain '%s'", expLine2)
    }

    expLine3 := fmt.Sprintf("%d\t%s\t%s\t%s\t", activity_2.Id, activity_2.Name,
      activity_2.Project, activity_2.Status())
    if !strings.Contains(cmdOutput, expLine3) {
      t.Errorf("expected output to contain '%s'", expLine3)
    }
  }
  DbTestRun(f, t)
}

func TestStatusCommand_Help(t *testing.T) {
  c := StatusCommand{}
  if c.Help() == "" {
    t.Error("no help available")
  }
}
