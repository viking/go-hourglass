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
  startHelp = "Usage: %s start <name> [project] [tag1[, tag2[, ...]]]\n\nStart a new activity"
  stopHelp = "Usage: %s stop\n\nStop all activities"
  listHelp = "Usage: %s list [all|week]\n\nList activities"
  editHelp = "Usage: %s edit <id> <name|project|tags|start|end> [value1[, [value2][, ...]]]\n\nEdit an activity\n\nFor the tags option, each tag should be a separate argument. Acceptable date formats are:\n\t2006-01-02 15:04\n\t2006-01-02 15:04 -0700"
  restartHelp = "Usage: %s restart <id>\n\nStart a new activity with all of the same values as another activity"
  deleteHelp = "Usage: %s delete <id>\n\nDelete an activity"
)

/* edit date format */
const (
  DateFormat = "2006-01-02 15:04"
  DateWithZoneFormat = "2006-01-02 15:04 -0700"
  TimeFormat = "15:04"
)

/* syntax error */
type SyntaxError string
func (s SyntaxError) Error() string {
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
    err = SyntaxError("missing name argument")
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
  return startHelp
}

/* restart */
type RestartCommand struct{}

func (RestartCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  if len(args) == 0 {
    err = SyntaxError("missing id argument")
    return
  }
  var id int64
  id, err = strconv.ParseInt(args[0], 10, 64)
  if err != nil {
    err = SyntaxError("invalid id argument")
    return
  }

  var activity *Activity
  activity, err = db.FindActivity(id)
  if err != nil {
    return
  }
  activity.Id = 0
  activity.Start = c.Now()
  activity.End = time.Time{}
  err = db.SaveActivity(activity)
  if err == nil {
    output = fmt.Sprintf("restarted activity %d (new id: %d)", id, activity.Id)
  }
  return
}

func (RestartCommand) Help() string {
  return restartHelp
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
  return stopHelp
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

func (cmd ListCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
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
      return
    } else {
      table := &activityTable{activities, c, tableModeDay}
      output = table.String()
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

        /* collect the day's activities */
        lower := i
        upper := i
        for ; i < len(activities) && activities[i].Start.Weekday() == day; i++ {
          upper++
        }
        table := &activityTable{activities[lower:upper], c, tableModeWeek}
        output += table.String()

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
      table := &activityTable{activities, c, tableModeAll}
      output = table.String()
    }
  }

  return
}

func (ListCommand) Help() string {
  return listHelp
}

type tableMode int
const (
  tableModeDay tableMode = iota
  tableModeWeek
  tableModeAll
)

type activityTable struct {
  activities []*Activity
  c Clock
  mode tableMode
}

func (table *activityTable) header() (output string) {
  switch table.mode {
  case tableModeDay, tableModeWeek:
    output = "| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|"
  case tableModeAll:
    output = "| date\t| id\t| name\t| project\t| tags\t| state\t| start\t| end\t| duration\t|"
  }
  return
}

func (table *activityTable) formatActivity(activity *Activity) (output string) {
  var date, start, end string
  if !activity.Start.IsZero() {
    date = activity.Start.Format("2006-01-02")
    start = activity.Start.Format(TimeFormat)
  }
  if !activity.End.IsZero() {
    end = activity.End.Format(TimeFormat)
  }
  duration := activity.Duration(table.c)
  output = fmt.Sprintf("| %d\t| %s\t| %s\t| %s\t| %s\t| %s\t| %s\t| %s\t|",
    activity.Id, activity.Name, activity.Project, activity.TagList(),
    activity.Status(), start, end, duration)
  if table.mode == tableModeAll {
    output = fmt.Sprintf("| %s\t%s", date, output)
  }
  return
}

func (table *activityTable) String() (output string) {
  totals := newProjectDurationList()
  output = table.header()
  for _, activity := range(table.activities) {
    output += fmt.Sprint("\n", table.formatActivity(activity))
    totals.add(activity.Project, activity.Duration(table.c))
  }
  if table.mode != tableModeAll {
    output += fmt.Sprint("\n", totals)
  }
  return
}

/* edit */
type EditCommand struct{}

func (EditCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  if len(args) > 1 {
    var id int64
    id, err = strconv.ParseInt(args[0], 10, 64)
    if err != nil {
      err = SyntaxError("non-integer id")
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
        err = SyntaxError("name is required")
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
          err = SyntaxError("invalid date")
          return
        }
        if args[1] == "start" {
          activity.Start = t
        } else {
          activity.End = t
        }
      } else {
        err = SyntaxError("date is required")
        return
      }
    default:
      err = SyntaxError("invalid field name")
      return
    }

    err = db.SaveActivity(activity)
    if err != nil {
      return
    }
    output = "ok"
  } else {
    err = SyntaxError("must have at least 3 arguments")
  }
  return
}

func (EditCommand) Help() string {
  return editHelp
}

/* delete */
type DeleteCommand struct{}

func (DeleteCommand) Run(c Clock, db Database, args ...string) (output string, err error) {
  if len(args) == 0 {
    err = SyntaxError("missing id argument")
    return
  }
  var id int64
  id, err = strconv.ParseInt(args[0], 10, 64)
  if err != nil {
    err = SyntaxError("invalid id argument")
    return
  }

  err = db.DeleteActivity(id)
  if err == nil {
    output = fmt.Sprint("deleted activity ", id)
  }
  return
}

func (DeleteCommand) Help() string {
  return deleteHelp
}
