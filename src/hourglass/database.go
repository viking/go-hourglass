package hourglass

import (
  "strings"
  "time"
  "errors"
)

var ErrNotFound = errors.New("record not found")

/* error helper */
type DatabaseErrors struct {
  Errors []string
}
func (e *DatabaseErrors) Error() string {
  return strings.Join(e.Errors, "; ")
}
func (e *DatabaseErrors) Append(err error) {
  e.Errors = append(e.Errors, err.Error())
}
func (e *DatabaseErrors) IsEmpty() bool {
  return len(e.Errors) == 0
}

/* main interface */
type Database interface {
  Valid() (bool, error)
  Version() (int, error)
  Migrate() error
  SaveActivity(*Activity) error
  FindActivity(id int64) (*Activity, error)
  FindAllActivities() ([]*Activity, error)
  FindRunningActivities() ([]*Activity, error)
  FindActivitiesBetween(time.Time, time.Time) ([]*Activity, error)
}
