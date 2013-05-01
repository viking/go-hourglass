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

func DbTestRun(f func (db *Sql), t *testing.T) {
  /* Create temporary file */
  dbFile, tempErr := ioutil.TempFile("", "hourglass")
  if tempErr != nil {
    t.Error(tempErr)
  }
  closeErr := dbFile.Close()
  if closeErr != nil {
    t.Error(closeErr)
  }

  db := &Sql{"sqlite", dbFile.Name(), nil}

  /* Check database validity, register driver if necessary */
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

  /* Migrate the database and run the function */
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

func TestDB_SaveActivity(t *testing.T) {
  f := func (db *Sql) {
    activity := &Activity{Name: "foo", Project: "bar"}
    activity.End = time.Now()
    activity.Start = activity.End.Add(-time.Hour)

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

func TestDB_SaveActivity_WithExistingActivity(t *testing.T) {
  f := func (db *Sql) {
    activity := &Activity{Name: "foo", Project: "bar"}
    activity.End = time.Now()
    activity.Start = activity.End.Add(-time.Hour)

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


func TestDB_FindActivity_WithNonExistantId(t *testing.T) {
  f := func(db *Sql) {
    _, findErr := db.FindActivity(1234)
    if findErr != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %T", findErr)
      return
    }
  }
  DbTestRun(f, t)
}

func TestDB_FindAllActivities(t *testing.T) {
  f := func (db *Sql) {
    activity := &Activity{Name: "foo", Project: "bar"}
    activity.End = time.Now()
    activity.Start = activity.End.Add(-time.Hour)

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

func TestDB_FindRunningActivities(t *testing.T) {
  f := func (db *Sql) {
    activity_1 := &Activity{Name: "foo", Project: "bar"}
    activity_1.End = time.Now()
    activity_1.Start = activity_1.End.Add(-time.Hour)

    activity_2 := &Activity{Name: "baz", Start: time.Now()}

    var saveErr error
    saveErr = db.SaveActivity(activity_1)
    if saveErr != nil {
      t.Error(saveErr)
    }
    saveErr = db.SaveActivity(activity_2)
    if saveErr != nil {
      t.Error(saveErr)
    }

    activities, findErr := db.FindRunningActivities()
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected to find 1 activity, but found", len(activities))
      return
    }

    if !activity_2.Equal(activities[0]) {
      t.Error("expected:\n", activity_2, "\ngot:\n", activities[0])
    }
  }
  DbTestRun(f, t)
}

func TestDB_FindActivitiesBetween(t *testing.T) {
  f := func (db *Sql) {
    now := time.Now()

    activity_1 := &Activity{Name: "foo", Project: "bar"}
    activity_1.End = now.Add(-(time.Hour * 24))
    activity_1.Start = activity_1.End.Add(-time.Hour)

    activity_2 := &Activity{Name: "baz", Start: now}

    var saveErr error
    saveErr = db.SaveActivity(activity_1)
    if saveErr != nil {
      t.Error(saveErr)
    }
    saveErr = db.SaveActivity(activity_2)
    if saveErr != nil {
      t.Error(saveErr)
    }

    activities, findErr := db.FindActivitiesBetween(activity_1.Start,
      activity_1.Start.Add(time.Hour))
    if findErr != nil {
      t.Error(findErr)
      return
    }

    if len(activities) != 1 {
      t.Error("expected to find 1 activity, but found", len(activities))
      return
    }

    if !activity_1.Equal(activities[0]) {
      t.Error("expected:\n", activity_1, "\ngot:\n", activities[0])
    }
  }
  DbTestRun(f, t)
}
