package commands

import (
  "time"
  "fmt"
  "sort"
  . "hourglass/database"
  . "hourglass/activity"
  . "hourglass/clock"
)

/* help messages */
const (
  StartHelp = "Usage: %s start <name> [project] [ [tag1, tag2, ...] ]\n\nStart a new activity"
  StopHelp = "Usage: %s stop\n\nStop all activities"
  StatusHelp = "Usage: %s status [all|week]\n\nShow activity status"
)

/* syntax error */
type SyntaxErr string
func (s SyntaxErr) Error() string {
  return fmt.Sprint("syntax error: ", string(s))
}

/* command interface */
type Command interface {
  Run(c Clock, db Database, args ...string) (string, error)
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

/* project duration, needed for sorting */
type projectDuration struct {
  name string
  duration Duration
}
type projectDurationList struct {
  slice []*projectDuration
}
func newProjectDurationList() *projectDurationList {
  pdl := &projectDurationList{}
  pdl.slice = make([]*projectDuration, 0, 10)
  return pdl
}
func (pdl *projectDurationList) Len() int {
  return len(pdl.slice)
}
func (pdl *projectDurationList) Less(i, j int) bool {
  if pdl.slice[i].name == "" {
    return false
  } else if pdl.slice[j].name == "" {
    return true
  }
  return pdl.slice[i].name < pdl.slice[j].name
}
func (pdl *projectDurationList) Swap(i, j int) {
  pdl.slice[i], pdl.slice[j] = pdl.slice[j], pdl.slice[i]
}
func (pdl *projectDurationList) add(name string, duration Duration) {
  var pd *projectDuration
  for _, val := range pdl.slice {
    if val.name == name {
      pd = val
      break
    }
  }
  if pd == nil {
    pdl.slice = append(pdl.slice, &projectDuration{name, duration})
    sort.Sort(pdl)
  } else {
    pd.duration += duration
  }
}
func (pdl *projectDurationList) String() (str string) {
  for i, pd := range pdl.slice {
    if i > 0 {
      str += ", "
    }
    var name string
    if pd.name == "" {
      name = "unsorted"
    } else {
      name = pd.name
    }
    str += fmt.Sprint(name, ": ", pd.duration)
  }
  return
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

    totals := newProjectDurationList()
    if len(activities) == 0 {
      output = "there have been no activities today"
    } else {
      output = fmt.Sprint("| id\t| name\t| project\t| tags\t| state\t| duration")
      for _, activity := range(activities) {
        duration := activity.Duration(c)
        output += fmt.Sprintf("\n| %d\t| %s\t| %s\t| %s\t| %s\t| %s",
          activity.Id, activity.Name, activity.Project, activity.TagList(),
          activity.Status(), duration)
        totals.add(activity.Project, duration)
      }
      output += fmt.Sprint("\n", totals)
    }
  } else if args[0] == "week" {
    now := c.Now()

    /* midnight Sunday */
    /* NOTE: zero and negative days work just fine here */
    lower := time.Date(now.Year(), now.Month(),
      now.Day() - int(now.Weekday()), 0, 0, 0, 0, now.Location())

    /* midnight Sunday next week */
    upper := time.Date(now.Year(), now.Month(),
      now.Day() + (7 - int(now.Weekday())), 0, 0, 0, 0, now.Location())

    var activities []*Activity
    activities, err = db.FindActivitiesBetween(lower, upper)
    if err != nil {
      return
    }

    if len(activities) == 0 {
      output = "there have been no activities this week"
    } else {
      numDays := 0
      for i, day := 0, time.Sunday; i < len(activities) && day <= time.Saturday; day++ {
        if activities[i].Start.Weekday() != day {
          /* don't print out day if there are no activities */
          continue
        }

        /* print out header for the day */
        date := time.Date(now.Year(), now.Month(),
          now.Day() - (int(now.Weekday()) - int(day)), 0, 0, 0, 0,
          now.Location())
        if numDays > 0 {
          output += "\n\n"
        }
        output += fmt.Sprintf("=== %s (%04d-%02d-%02d) ===\n",
          day, date.Year(), int(date.Month()), date.Day())
        output += fmt.Sprint("| id\t| name\t| project\t| tags\t| state\t| duration")

        /* print out the day's activities */
        totals := newProjectDurationList()
        for ; i < len(activities) && activities[i].Start.Weekday() == day; i++ {
          activity := activities[i]

          duration := activity.Duration(c)
          output += fmt.Sprintf("\n| %d\t| %s\t| %s\t| %s\t| %s\t| %s",
            activity.Id, activity.Name, activity.Project, activity.TagList(),
            activity.Status(), duration)
          totals.add(activity.Project, duration)
        }
        output += fmt.Sprint("\n", totals)

        numDays++
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
