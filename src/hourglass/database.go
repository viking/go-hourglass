package hourglass

import (
  "database/sql"
  "strings"
  "time"
  "errors"
)

const DatabaseVersion = 2

type Database struct {
  DriverName string
  DataSourceName string
}

type DatabaseErrors struct {
  Errors []string
}

var ErrNotFound = errors.New("record not found")

func (e *DatabaseErrors) Error() string {
  return strings.Join(e.Errors, "; ")
}

func (e *DatabaseErrors) Append(err error) {
  e.Errors = append(e.Errors, err.Error())
}

func (e *DatabaseErrors) IsEmpty() bool {
  return len(e.Errors) == 0
}

func (db *Database) Valid() (bool, error) {
  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    return false, openErr
  }

  connErr := conn.Close()
  if connErr != nil {
    return false, connErr
  }
  return true, nil
}

func (db *Database) Migrate() error {
  err := &DatabaseErrors{}

  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    err.Append(openErr)
    return err
  }

  versionRow := conn.QueryRow("SELECT version FROM schema_info")
  version := 0
  versionRow.Scan(&version)

  var execErr error
  for ; version < DatabaseVersion; version++ {
    switch version {
    case 0:
      _, execErr = conn.Exec(`CREATE TABLE schema_info (version INT)`)
    case 1:
      _, execErr = conn.Exec(`CREATE TABLE activities (id INTEGER PRIMARY KEY,
        name TEXT, project TEXT, tags TEXT, start TIMESTAMP, end TIMESTAMP)`)
    }

    if execErr != nil {
      err.Append(execErr)
      break
    } else {
      _, execErr = conn.Exec("INSERT INTO schema_info VALUES(?)", version + 1)
      if execErr != nil {
        err.Append(execErr)
        break
      }
    }
  }

  connErr := conn.Close()
  if connErr != nil {
    err.Append(connErr)
  }

  if err.IsEmpty() {
    return nil
  }
  return err
}

func (db *Database) SaveActivity(a *Activity) error {
  err := &DatabaseErrors{}

  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    err.Append(openErr)
    return err
  }

  var query string
  var args []interface{}
  if (a.Id == 0) {
    query = `
      INSERT INTO activities (name, project, tags, start, end)
      VALUES(?, ?, ?, ?, ?)
    `
    args = []interface{}{a.Name, a.Project, a.TagList(), a.Start, a.End}
  } else {
    query = `
      UPDATE activities SET name = ?, project = ?, tags = ?,
      start = ?, end = ? WHERE id = ?
    `
    args = []interface{}{a.Name, a.Project, a.TagList(), a.Start, a.End, a.Id}
  }

  /* Execute the query */
  res, execErr := conn.Exec(query, args...)
  if execErr == nil {
    if a.Id == 0 {
      id, idErr := res.LastInsertId()
      if idErr == nil {
        a.Id = id
      } else {
        err.Append(idErr)
      }
    }
  } else {
    err.Append(execErr)
  }

  connErr := conn.Close()
  if connErr != nil {
    err.Append(connErr)
  }

  if err.IsEmpty() {
    return nil
  }
  return err
}

func (db *Database) FindActivities(predicate string, args ...interface{}) ([]*Activity, error) {
  var activities []*Activity = nil
  err := &DatabaseErrors{}

  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    err.Append(openErr)
    return activities, err
  }

  query := `SELECT id, name, project, tags, start, end
    FROM activities ` + predicate
  rows, queryErr := conn.Query(query, args...)

  if queryErr != nil {
    err.Append(queryErr)
  } else {
    for rows.Next() {
      var id int64
      var name, project, tagList string
      var start, end time.Time

      scanErr := rows.Scan(&id, &name, &project, &tagList, &start, &end)
      if scanErr == nil {
        activity := &Activity{Id: id, Name: name, Project: project, Start: start, End: end}
        activity.SetTagList(tagList)
        activities = append(activities, activity)
      } else {
        err.Append(scanErr)
      }
    }
  }

  connErr := conn.Close()
  if connErr != nil {
    err.Append(connErr)
  }

  if err.IsEmpty() {
    return activities, nil
  }
  return activities, err
}

func (db *Database) FindActivity(id int64) (*Activity, error) {
  activities, findErr := db.FindActivities("WHERE id = ?", id)
  if findErr != nil {
    return nil, findErr
  }
  if len(activities) == 0 {
    return nil, ErrNotFound
  }
  return activities[0], nil
}

func (db *Database) FindAllActivities() (activities []*Activity, err error) {
  activities, err = db.FindActivities("")
  return
}

func (db *Database) FindRunningActivities() (activities []*Activity, err error) {
  activities, err = db.FindActivities("WHERE end IS ?", &time.Time{})
  return
}

