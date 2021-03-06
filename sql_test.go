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

func sqlTestRun(f func (db *Sql), t *testing.T) {
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

func TestSql_Version(t *testing.T) {
  f := func (db *Sql) {
    version, err := db.Version()
    if err != nil {
      t.Error(err)
    } else if version != SqlVersion {
      t.Errorf("expected version to be %d, but was %d", SqlVersion, version)
    }
  }
  sqlTestRun(f, t)
}

func TestSql_SaveActivity(t *testing.T) {
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
  sqlTestRun(f, t)
}

func TestSql_SaveActivity_WithExistingActivity(t *testing.T) {
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
  sqlTestRun(f, t)
}


func TestSql_FindActivity_WithNonExistantId(t *testing.T) {
  f := func(db *Sql) {
    _, findErr := db.FindActivity(1234)
    if findErr != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %T", findErr)
      return
    }
  }
  sqlTestRun(f, t)
}

func TestSql_FindAllActivities(t *testing.T) {
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
  sqlTestRun(f, t)
}

func TestSql_FindRunningActivities(t *testing.T) {
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
  sqlTestRun(f, t)
}

func TestSql_FindActivitiesBetween(t *testing.T) {
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
  sqlTestRun(f, t)
}

func TestSql_DeleteActivity(t *testing.T) {
  f := func(db *Sql) {
    var err error
    activity := &Activity{Name: "foo"}
    err = db.SaveActivity(activity)
    if err != nil {
      t.Error(err)
      return
    }

    err = db.DeleteActivity(activity.Id)
    if err != nil {
      t.Error(err)
    }

    _, err = db.FindActivity(activity.Id)
    if err != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %v", err)
    }
  }
  sqlTestRun(f, t)
}

func TestSql_DeleteActivity_WithBadId(t *testing.T) {
  f := func(db *Sql) {
    var err error

    err = db.DeleteActivity(123)
    if err != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %v", err)
    }
  }
  sqlTestRun(f, t)
}
