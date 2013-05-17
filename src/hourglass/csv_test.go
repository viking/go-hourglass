package hourglass

import (
  "testing"
  "io/ioutil"
  "os"
  "time"
)

func csvTestRun(f func (db *Csv), t *testing.T) {
  var err error

  /* Create temporary file */
  var csvFile *os.File
  csvFile, err = ioutil.TempFile("", "hourglass")
  if err != nil {
    t.Error(err)
  }
  err = csvFile.Close()
  if err != nil {
    t.Error(err)
  }
  /*fmt.Println(csvFile.Name())*/
  defer os.Remove(csvFile.Name())

  var db *Csv
  db, err = NewCsv(csvFile.Name())
  if err != nil {
    t.Fatal(err)
  }

  /* Migrate the database and run the function */
  err = db.Migrate()
  if err != nil {
    t.Error(err)
  } else {
    f(db)
  }
}

func TestCsv_Version(t *testing.T) {
  f := func (db *Csv) {
    version, err := db.Version()
    if err != nil {
      t.Error(err)
    } else if version != CsvVersion {
      t.Errorf("expected version to be %d, but was %d", CsvVersion, version)
    }
  }
  csvTestRun(f, t)
}

func TestCsv_SaveActivity(t *testing.T) {
  f := func (db *Csv) {
    activity := &Activity{Name: "foo", Project: "bar"}
    activity.End = time.Now()
    activity.Start = activity.End.Add(-time.Hour)

    var err error
    err = db.SaveActivity(activity)
    if err != nil {
      t.Error(err)
      return
    }
    if activity.Id == 0 {
      t.Error("expected activity.Id to be non-zero")
      return
    }

    var foundActivity *Activity
    foundActivity, err = db.FindActivity(activity.Id)
    if err != nil {
      t.Error(err)
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
  csvTestRun(f, t)
}

var updateTests = []struct {
  numAfter int
  original *Activity
  changes *Activity
}{
  {0, &Activity{Name: "foo", Project: "bar"}, &Activity{Name: "baz", Project: "bar"}},
  {1, &Activity{Name: "foo", Project: "bar"}, &Activity{Name: "baz", Project: "bar"}},
  {0, &Activity{Name: "foo", Project: "bar"}, &Activity{Name: "blah", Project: "quux"}},
  {1, &Activity{Name: "foo", Project: "bar"}, &Activity{Name: "blah", Project: "quux"}},
}

func TestCsv_SaveActivity_WithExistingActivity(t *testing.T) {
  for testNum, config := range updateTests {
    f := func (db *Csv) {
      var err error
      activities := make([]*Activity, config.numAfter + 1)
      for i := range activities {
        var activity *Activity
        if i == 0 {
          activity = config.original.Clone()
        } else {
          activity = &Activity{Name: "junk", Project: "test"}
        }
        activities[i] = activity

        err = db.SaveActivity(activity)
        if err != nil {
          t.Errorf("test %d: %s", testNum, err)
          return
        }
      }

      activities[0].Name = config.changes.Name
      activities[0].Project = config.changes.Project
      err = db.SaveActivity(activities[0])
      if err != nil {
        t.Errorf("test %d: %s", testNum, err)
        return
      }

      for i, activity := range activities {
        id := int64(i + 1)
        var foundActivity *Activity
        foundActivity, err = db.FindActivity(id)
        if err != nil {
          t.Errorf("test %d: %s", testNum, err)
        } else if foundActivity == nil {
          t.Errorf("test %d: couldn't find activity %d", testNum, id)
        } else if !activity.Equal(foundActivity) {
          t.Errorf("test %d: expected:\n%v\ngot:\n%v", testNum,
            activity, foundActivity)
        }
      }
    }
    csvTestRun(f, t)
  }
}

func TestCsv_FindAllActivities(t *testing.T) {
  f := func (db *Csv) {
    activity_1 := &Activity{Name: "foo", Project: "bar"}
    activity_1.End = time.Now()
    activity_1.Start = activity_1.End.Add(-time.Hour)
    err := db.SaveActivity(activity_1)
    if err != nil {
      t.Error(err)
      return
    }

    activity_2 := &Activity{Name: "baz", Project: "blargh"}
    activity_2.End = time.Now()
    activity_2.Start = activity_2.End.Add(-time.Hour)
    err = db.SaveActivity(activity_2)
    if err != nil {
      t.Error(err)
      return
    }

    var activities []*Activity
    activities, err = db.FindAllActivities()
    if err != nil {
      t.Error(err)
      return
    }

    if len(activities) != 2 {
      t.Error("expected to find 2 activity, but found", len(activities))
      return
    }

    if !activity_1.Equal(activities[0]) {
      t.Error("expected:\n", activity_1, "\ngot:\n", activities[0])
    }

    if !activity_2.Equal(activities[1]) {
      t.Error("expected:\n", activity_2, "\ngot:\n", activities[1])
    }
  }
  csvTestRun(f, t)
}

func TestCsv_FindRunningActivities(t *testing.T) {
  f := func (db *Csv) {
    activity_1 := &Activity{Name: "foo", Project: "bar"}
    activity_1.End = time.Now()
    activity_1.Start = activity_1.End.Add(-time.Hour)

    activity_2 := &Activity{Name: "baz", Start: time.Now()}

    err := db.SaveActivity(activity_1)
    if err != nil {
      t.Error(err)
    }
    err = db.SaveActivity(activity_2)
    if err != nil {
      t.Error(err)
    }

    var activities []*Activity
    activities, err = db.FindRunningActivities()
    if err != nil {
      t.Error(err)
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
  csvTestRun(f, t)
}

func TestCsv_FindActivitiesBetween(t *testing.T) {
  f := func (db *Csv) {
    now := time.Now()

    activity_1 := &Activity{Name: "foo", Project: "bar"}
    activity_1.End = now.Add(-(time.Hour * 24))
    activity_1.Start = activity_1.End.Add(-time.Hour)

    activity_2 := &Activity{Name: "baz", Start: now}

    err := db.SaveActivity(activity_1)
    if err != nil {
      t.Error(err)
    }
    err = db.SaveActivity(activity_2)
    if err != nil {
      t.Error(err)
    }

    var activities []*Activity
    activities, err = db.FindActivitiesBetween(activity_1.Start,
      activity_1.Start.Add(time.Hour))
    if err != nil {
      t.Error(err)
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
  csvTestRun(f, t)
}

func TestCsv_DeleteActivity(t *testing.T) {
  f := func(db *Csv) {
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
  csvTestRun(f, t)
}

func TestCsv_DeleteActivity_FromMiddle(t *testing.T) {
  f := func(db *Csv) {
    var err error
    activity_1 := &Activity{Name: "foo"}
    err = db.SaveActivity(activity_1)
    if err != nil {
      t.Error(err)
      return
    }

    activity_2 := &Activity{Name: "bar"}
    err = db.SaveActivity(activity_2)
    if err != nil {
      t.Error(err)
      return
    }

    err = db.DeleteActivity(activity_1.Id)
    if err != nil {
      t.Error(err)
    }

    _, err = db.FindActivity(activity_1.Id)
    if err != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %v", err)
    }

    var foundActivity_2 *Activity
    foundActivity_2, err = db.FindActivity(activity_2.Id)
    if err != nil {
      t.Error(err)
    } else if !activity_2.Equal(foundActivity_2) {
      t.Errorf("expected %v, got %v", activity_2, foundActivity_2)
    }
  }
  csvTestRun(f, t)
}

func TestCsv_DeleteActivity_WithBadId(t *testing.T) {
  f := func(db *Csv) {
    var err error

    err = db.DeleteActivity(123)
    if err != ErrNotFound {
      t.Errorf("expected ErrNotFound, got %v", err)
    }
  }
  csvTestRun(f, t)
}
