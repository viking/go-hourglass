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
