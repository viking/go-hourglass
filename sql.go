package hourglass

import (
  "database/sql"
  "io"
  "fmt"
  "time"
)

const SqlVersion = 2

/* sql backend */
type Sql struct {
  DriverName string
  DataSourceName string
  Log io.Writer
}

func (db *Sql) exec(conn *sql.DB, query string, args ...interface{}) (res sql.Result, err error) {
  if db.Log != nil {
    message := fmt.Sprintf("exec: \"%s\" with args: %v\n", query, args)
    db.Log.Write([]byte(message))
  }
  res, err = conn.Exec(query, args...)
  return
}

func (db *Sql) query(conn *sql.DB, query string, args ...interface{}) (rows *sql.Rows, err error) {
  if db.Log != nil {
    message := fmt.Sprintf("query: \"%s\" with args: %v\n", query, args)
    db.Log.Write([]byte(message))
  }
  rows, err = conn.Query(query, args...)
  return
}

func (db *Sql) queryRow(conn *sql.DB, query string, args ...interface{}) (row *sql.Row) {
  if db.Log != nil {
    message := fmt.Sprintf("queryRow: \"%s\" with args: %v\n", query, args)
    db.Log.Write([]byte(message))
  }
  row = conn.QueryRow(query, args...)
  return
}

func (db *Sql) Valid() (bool, error) {
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

func (db *Sql) Version() (version int, err error) {
  var conn *sql.DB

  conn, err = sql.Open(db.DriverName, db.DataSourceName)
  if err != nil {
    return
  }

  versionRow := db.queryRow(conn, "SELECT version FROM schema_info")
  versionRow.Scan(&version)
  return
}

func (db *Sql) Migrate() error {
  err := &DatabaseErrors{}

  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    err.Append(openErr)
    return err
  }

  versionRow := db.queryRow(conn, "SELECT version FROM schema_info")
  version := 0
  versionRow.Scan(&version)

  var execErr error
  for ; version < SqlVersion; version++ {
    switch version {
    case 0:
      _, execErr = db.exec(conn, `CREATE TABLE schema_info (version INT)`)
      if execErr == nil {
        _, execErr = db.exec(conn, "INSERT INTO schema_info VALUES (?)", 0)
      }
    case 1:
      _, execErr = db.exec(conn, `CREATE TABLE activities (id INTEGER PRIMARY KEY,
        name TEXT, project TEXT, tags TEXT, start TIMESTAMP, end TIMESTAMP)`)
    }

    if execErr != nil {
      err.Append(execErr)
      break
    } else {
      _, execErr = db.exec(conn, "UPDATE schema_info SET version = ?", version + 1)
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

func (db *Sql) SaveActivity(a *Activity) error {
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
    args = []interface{}{a.Name, a.Project, a.TagList(), a.Start.UTC(), a.End.UTC()}
  } else {
    query = `
      UPDATE activities SET name = ?, project = ?, tags = ?,
      start = ?, end = ? WHERE id = ?
    `
    args = []interface{}{a.Name, a.Project, a.TagList(), a.Start.UTC(), a.End.UTC(), a.Id}
  }

  /* Execute the query */
  res, execErr := db.exec(conn, query, args...)
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

func (db *Sql) findActivities(predicate string, args ...interface{}) ([]*Activity, error) {
  var activities []*Activity = nil
  err := &DatabaseErrors{}

  conn, openErr := sql.Open(db.DriverName, db.DataSourceName)
  if openErr != nil {
    err.Append(openErr)
    return activities, err
  }

  query := `SELECT id, name, project, tags, start, end
    FROM activities ` + predicate
  rows, queryErr := db.query(conn, query, args...)

  if queryErr != nil {
    err.Append(queryErr)
  } else {
    for rows.Next() {
      var id int64
      var name, project, tagList string
      var start, end time.Time

      scanErr := rows.Scan(&id, &name, &project, &tagList, &start, &end)
      if scanErr == nil {
        activity := &Activity{Id: id, Name: name, Project: project, Start: start.Local(), End: end.Local()}
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

func (db *Sql) FindActivity(id int64) (*Activity, error) {
  activities, findErr := db.findActivities("WHERE id = ?", id)
  if findErr != nil {
    return nil, findErr
  }
  if len(activities) == 0 {
    return nil, ErrNotFound
  }
  return activities[0], nil
}

func (db *Sql) FindAllActivities() (activities []*Activity, err error) {
  activities, err = db.findActivities("")
  return
}

func (db *Sql) FindRunningActivities() (activities []*Activity, err error) {
  activities, err = db.findActivities("WHERE end IS ?", &time.Time{})
  return
}

func (db *Sql) FindActivitiesBetween(lower time.Time, upper time.Time) (activities []*Activity, err error) {
  activities, err = db.findActivities("WHERE start >= ? AND start < ?", lower, upper)
  return
}

func (db *Sql) DeleteActivity(id int64) (err error) {
  var conn *sql.DB
  conn, err = sql.Open(db.DriverName, db.DataSourceName)
  if err != nil {
    return
  }
  defer conn.Close()

  var result sql.Result
  result, err = db.exec(conn, "DELETE FROM activities WHERE id = ?", id)
  if err == nil {
    var n int64
    n, err = result.RowsAffected()
    if err == nil && n != 1 {
      err = ErrNotFound
    }
  }
  return
}
