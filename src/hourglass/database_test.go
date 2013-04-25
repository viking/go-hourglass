package hourglass

import (
  "testing"
  "io/ioutil"
  "os"
  "time"
  "strings"
  "database/sql"
  sqlite "github.com/mattn/go-sqlite3"
)

func DbTestRun(f func (db *Database), t *testing.T) {
  dbFile, tempErr := ioutil.TempFile("", "hourglass")
  if tempErr != nil {
    t.Error(tempErr)
  }
  closeErr := dbFile.Close()
  if closeErr != nil {
    t.Error(closeErr)
  }

  db := &Database{"sqlite", dbFile.Name()}

  var ok bool
  var dbErr error
  if ok, dbErr = db.Valid(); !ok {
    if strings.Contains(dbErr.Error(), "unknown driver") {
      sql.Register("sqlite", &sqlite.SQLiteDriver{})
      ok = true
    } else {
      t.Error(dbErr)
    }
  }

  if ok {
    migrateErr := db.Migrate()
    if migrateErr != nil {
      t.Error(migrateErr)
    } else {
      f(db)
    }
  }

  /*t.Log(dbFile.Name())*/
  os.Remove(dbFile.Name())
}

func TestDatabase_SaveActivity(t *testing.T) {
  activity := &Activity{Name: "foo", Project: "bar"}
  activity.End = time.Now().UTC()
  activity.Start = activity.End.Add(-time.Hour)

  f := func (db *Database) {
    saveErr := db.SaveActivity(activity)
    if saveErr != nil {
      t.Error(saveErr)
      return
    }
    if activity.Id == 0 {
      t.Error("expected activity.Id to be non-zero")
      return
    }

    foundActivity, findErr := db.FindActivity(activity.Id)
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if foundActivity == nil {
      t.Error("couldn't find activity")
      return
    }

    if !activity.Equal(foundActivity) {
      t.Error("expected:\n", activity, "\ngot:\n", foundActivity)
    }
  }
  DbTestRun(f, t)
}

func TestDatabase_SaveActivity_WithExistingActivity(t *testing.T) {
  activity := &Activity{Name: "foo", Project: "bar"}
  activity.End = time.Now().UTC()
  activity.Start = activity.End.Add(-time.Hour)

  f := func (db *Database) {
    saveErr := db.SaveActivity(activity)
    if saveErr != nil {
      t.Error(saveErr)
      return
    }

    activity.Name = "baz"
    saveErr = db.SaveActivity(activity)
    if saveErr != nil {
      t.Error(saveErr)
      return
    }

    foundActivity, findErr := db.FindActivity(activity.Id)
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if foundActivity == nil {
      t.Error("couldn't find activity")
      return
    }

    if !activity.Equal(foundActivity) {
      t.Error("expected:\n", activity, "\ngot:\n", foundActivity)
    }
  }
  DbTestRun(f, t)
}

func TestDatabase_FindAllActivities(t *testing.T) {
  activity := &Activity{Name: "foo", Project: "bar"}
  activity.End = time.Now().UTC()
  activity.Start = activity.End.Add(-time.Hour)

  f := func (db *Database) {
    saveErr := db.SaveActivity(activity)
    if saveErr != nil {
      t.Error(saveErr)
      return
    }

    activities, findErr := db.FindAllActivities()
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected to find 1 activity, but found", len(activities))
      return
    }

    if !activity.Equal(activities[0]) {
      t.Error("expected:\n", activity, "\ngot:\n", activities[0])
    }
  }
  DbTestRun(f, t)
}
