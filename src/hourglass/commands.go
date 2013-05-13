package hourglass

import (
  "time"
  "fmt"
  "sort"
  "strconv"
  "strings"
)

/* help messages */
const (
  StartHelp = "Usage: %s start <name> [project] [tag1[, tag2[, ...]]]\n\nStart a new activity"
  StopHelp = "Usage: %s stop\n\nStop all activities"
  ListHelp = "Usage: %s list [all|week]\n\nList activities"
  EditHelp = "Usage: %s edit <id> <name|project|tags|start|end> [value1[, [value2][, ...]]]\n\nEdit an activity\n\nFor the tags option, each tag should be a separate argument. Acceptable date formats are:\n\t2006-01-02 15:04\n\t2006-01-02 15:04 -0700"
)

/* edit date format */
const (
  DateFormat = "2006-01-02 15:04"
  DateWithZoneFormat = "2006-01-02 15:04 -0700"
)

/* syntax error */
type ErrSyntax string
func (s ErrSyntax) Error() string {
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
    err = ErrSyntax("missing name argument")
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

/* list */
type ListCommand struct{}

func (ListCommand) Run(c Clock, db Database, args ...string) (output string, err error) {

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

func (ListCommand) Help() string {
  return ListHelp
}

/* edit */
type EditCommand struct{}

func (EditCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  if len(args) > 1 {
    var id int64
    id, err = strconv.ParseInt(args[0], 10, 64)
    if err != nil {
      err = ErrSyntax("non-integer id")
      return
    }

    var activity *Activity
    activity, err = db.FindActivity(id)
    if err != nil {
      return
    }

    field := args[1]
    switch(field) {
    case "name":
      if len(args) > 2 {
        activity.Name = strings.Join(args[2:], " ")
      } else {
        err = ErrSyntax("name is required")
        return
      }
    case "project":
      if len(args) > 2 {
        activity.Project = strings.Join(args[2:], " ")
      } else {
        activity.Project = ""
      }
    case "tags":
      activity.Tags = args[2:]
    case "start", "end":
      if len(args) > 2 {
        dateString := strings.Join(args[2:], " ")

        var t time.Time
        t, err = time.ParseInLocation(DateFormat, dateString, time.Local)
        if err != nil {
          t, err = time.Parse(DateWithZoneFormat, dateString)
        }
        if err != nil {
          err = ErrSyntax("invalid date")
          return
        }
        if args[1] == "start" {
          activity.Start = t
        } else {
          activity.End = t
        }
      } else {
        err = ErrSyntax("date is required")
        return
      }
    default:
      err = ErrSyntax("invalid field name")
      return
    }

    err = db.SaveActivity(activity)
    if err != nil {
      return
    }
    output = "ok"
  } else {
    err = ErrSyntax("must have at least 3 arguments")
  }
  return
}

func (EditCommand) Help() string {
  return EditHelp
}
