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
  t.Log(csvFile.Name())
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

func TestCsv_ActivityRoundTrip(t *testing.T) {
  f := func (db *Csv) {
    activity := &Activity{Name: "foo", Project: "bar"}
    activity.End = time.Now()
    activity.Start = activity.End.Add(-time.Hour)

    var err error
    err = db.SaveActivity(activity)
    if err != nil {
      t.Fatal(err)
    }
    if activity.Id == 0 {
      t.Fatal("expected activity.Id to be non-zero")
    }

    var foundActivity *Activity
    foundActivity, err = db.FindActivity(activity.Id)
    if err != nil {
      t.Fatal(err)
    }

    if foundActivity == nil {
      t.Fatal("couldn't find activity")
    }

    if !activity.Equal(foundActivity) {
      t.Error("expected:\n", activity, "\ngot:\n", foundActivity)
    }
  }
  csvTestRun(f, t)
}
