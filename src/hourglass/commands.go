package hourglass

import (
  "time"
  "errors"
  "fmt"
)

const (
  StartHelp = "Usage: %s start <name> [project] [ [tag1, tag2, ...] ]\n\nStart a new activity"
  StopHelp = "Usage: %s stop\n\nStop all activities"
  StatusHelp = "Usage: %s status\n\nShow activity status"
)

type Command interface {
  Run(db Database, args ...string) (string, error)
  Help() string
}

/* start */
type StartCommand struct{}

func (StartCommand) Run(db Database, args ...string) (output string, err error) {
  var name, project string
  var tags []string

  if len(args) == 0 {
    err = errors.New("Missing name argument")
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
    output = fmt.Sprintf("started activity %d", activity.Id)
  }
  return
}

func (StartCommand) Help() string {
  return StartHelp
}

/* stop */
type StopCommand struct{}

func (StopCommand) Run(db Database, args ...string) (output string, err error) {
  var activities []*Activity

  end := time.Now().UTC()
  if len(args) == 0 {
    activities, err = db.FindRunningActivities()
    if err != nil {
      return
    }
    for i, activity := range activities {
      activity.End = end
      err = db.SaveActivity(activity)
      if err != nil {
        return
      }
      if i > 0 {
        output += "\n"
      }
      output += fmt.Sprintf("stopped activity %d", activity.Id)
    }
  }

  return
}

func (StopCommand) Help() string {
  return StopHelp
}

/* status */
type StatusCommand struct{}

func (StatusCommand) Run(db Database, args ...string) (output string, err error) {
  now := time.Now()

  /* midnight today */
  lower := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC()
  /* midnight tomorrow */
  upper := lower.Add(time.Hour * 24)

  var activities []*Activity
  activities, err = db.FindActivitiesBetween(lower, upper)
  if err != nil {
    return
  }

  if len(activities) == 0 {
    output = "there have been no activities today"
  } else {
    output = fmt.Sprint("id\tname\tproject\ttags\tstate\tduration")
    for _, activity := range(activities) {
      output += fmt.Sprintf("\n%d\t%s\t%s\t%s\t%s\t%s", activity.Id, activity.Name,
          activity.Project, activity.TagList(), activity.Status(),
          activity.Duration())
    }
  }

  return
}

func (StatusCommand) Help() string {
  return StatusHelp
}
