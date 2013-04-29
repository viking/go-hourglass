package commands

import (
  "time"
  "fmt"
  . "hourglass/database"
  . "hourglass/activity"
  . "hourglass/clock"
)

const (
  StartHelp = "Usage: %s start <name> [project] [ [tag1, tag2, ...] ]\n\nStart a new activity"
  StopHelp = "Usage: %s stop\n\nStop all activities"
  StatusHelp = "Usage: %s status [all]\n\nShow activity status"
)

type SyntaxErr string

func (s SyntaxErr) Error() string {
  return fmt.Sprint("syntax error: ", string(s))
}

type Command interface {
  Run(db Database, args ...string) (string, error)
  Help() string
}

/* start */
type StartCommand struct{}

func (StartCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  var name, project string
  var tags []string

  if len(args) == 0 {
    err = SyntaxErr("missing name argument")
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
    Start: c.Now(),
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

func (StopCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  var activities []*Activity

  end := c.Now()
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

func (StatusCommand) Run(c Clock, db Database, args ...string) (output string, err error) {

  if len(args) == 0 {
    now := c.Now()

    /* midnight today */
    lower := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
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
      output = fmt.Sprint("| id\t| name\t| project\t| tags\t| state\t| duration")
      for _, activity := range(activities) {
        output += fmt.Sprintf("\n| %d\t| %s\t| %s\t| %s\t| %s\t| %s",
          activity.Id, activity.Name, activity.Project, activity.TagList(),
          activity.Status(), activity.Duration(c))
      }
    }
  } else if args[0] == "all" {
    var activities []*Activity
    activities, err = db.FindAllActivities()
    if err != nil {
      return
    }

    if len(activities) == 0 {
      output = "there aren't any activities"
    } else {
      output = fmt.Sprint("| date\t| id\t| name\t| project\t| tags\t| state\t| duration")
      for _, activity := range(activities) {
        output += fmt.Sprintf("\n| %04d-%02d-%02d\t| %d\t| %s\t| %s\t| %s\t| %s\t| %s",
          activity.Start.Year(), activity.Start.Month(), activity.Start.Day(),
          activity.Id, activity.Name, activity.Project, activity.TagList(),
          activity.Status(), activity.Duration(c))
      }
    }
  }

  return
}

func (StatusCommand) Help() string {
  return StatusHelp
}
