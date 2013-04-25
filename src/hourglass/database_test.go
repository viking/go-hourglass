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

func dbRun(f func (db *Database), t *testing.T) {
  dbFile, err := ioutil.TempFile("", "hourglass")
  if err != nil {
    t.Error(err)
  }
  dbFile.Close()

  db := &Database{"sqlite", dbFile.Name()}
  if ok, dbErr := db.Valid(); !ok {
    if strings.Contains(dbErr.Error(), "unknown driver") {
      sql.Register("sqlite", &sqlite.SQLiteDriver{})
    } else {
      t.Error(dbErr)
    }
  } else {
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

func TestNewActivityRoundTrip(t *testing.T) {
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

    gotActivity, getErr := db.GetActivity(activity.Id)
    if getErr != nil {
      t.Error(getErr)
      return
    }

    if gotActivity == nil {
      t.Error("couldn't get activity")
      return
    }

    if !activity.Equal(gotActivity) {
      t.Error("expected:\n", activity, "\ngot:\n", gotActivity)
    }
  }
  dbRun(f, t)
}

func TestSaveExistingActivity(t *testing.T) {
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

    gotActivity, getErr := db.GetActivity(activity.Id)
    if getErr != nil {
      t.Error(getErr)
      return
    }

    if gotActivity == nil {
      t.Error("couldn't get activity")
      return
    }

    if !activity.Equal(gotActivity) {
      t.Error("expected:\n", activity, "\ngot:\n", gotActivity)
    }
  }
  dbRun(f, t)
}
