package hourglass

import (
  "time"
  "errors"
  "fmt"
)

const (
  StartHelp = "Usage: %s start <name> [project] [ [tag1, tag2, ...] ]\n\n" +
    "Start a new activity"
  StopHelp = "Usage: %s stop\n\nStop all activities"
)

type Command interface {
  Run(db *Database, args ...string) (string, error)
  Help() string
  NeedsDatabase() bool
}

/* start */
type StartCommand struct{}

func (StartCommand) Run(db *Database, args ...string) (output string, err error) {
  var name, project string
  var tags []string

  if len(args) == 0 {
    err = errors.New("missing name argument")
    return
  }

  for i, val := range args {
    switch i {
    case 0:
      name = val
    case 1:
      project = val
    case 2:
      tags = args[2:]
      break
    }
  }

  activity := &Activity{
    Name: name, Project: project, Tags: tags,
    Start: time.Now().UTC(),
  }
  err = db.SaveActivity(activity)
  if err == nil {
    output = fmt.Sprintf("started activity %d\n", activity.Id)
  }
  return
}

func (StartCommand) Help() string {
  return StartHelp
}

func (StartCommand) NeedsDatabase() bool {
  return true
}

/* stop */
type StopCommand struct{}

func (StopCommand) Run(db *Database, args ...string) (output string, err error) {
  var activities []*Activity

  end := time.Now().UTC()
  if len(args) == 0 {
    activities, err = db.FindRunningActivities()
    if err != nil {
      return
    }
    for _, activity := range activities {
      activity.End = end
      err = db.SaveActivity(activity)
      if err != nil {
        return
      }
      output += fmt.Sprintf("stopped activity %d\n", activity.Id)
    }
  }

  return
}

func (StopCommand) Help() string {
  return StopHelp
}

func (StopCommand) NeedsDatabase() bool {
  return true
}
