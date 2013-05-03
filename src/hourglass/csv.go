package hourglass

import (
  "encoding/csv"
  "os"
  "io"
  "bufio"
  "regexp"
  "strconv"
  "sync"
  "errors"
  "fmt"
  "time"
)

const CsvVersion = 1

var ErrBadFrontMatter = errors.New("invalid front matter")

type Csv struct {
  Filename string
  Mutex sync.RWMutex
  version int
  lastId int64
}

func NewCsv(filename string) (db *Csv, err error) {
  db = &Csv{Filename: filename}
  err = db.readFrontMatter()
  return
}

func (db *Csv) seekPastFrontMatter(f *os.File) (err error) {
  _, err = f.Seek(45, 0)
  return
}

func (db *Csv) readFrontMatter() (err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.Open(db.Filename)
  if err != nil {
    return
  }
  defer f.Close()

  r := bufio.NewReader(f)

  var line string
  line, err = r.ReadString('\n')
  if err != nil {
    if err == io.EOF && line == "" {
      /* This file is completely empty, so don't return an error */
      err = nil
    }
    return
  }

  var re *regexp.Regexp
  re, err = regexp.Compile("^# version: (\\d{3}), last-id: (\\d{19})\n")
  if err != nil {
    return
  }

  matches := re.FindStringSubmatch(line)
  if len(matches) != 3 {
    err = ErrBadFrontMatter
    return
  }

  db.version, err = strconv.Atoi(matches[1])
  if err != nil {
    err = ErrBadFrontMatter
    return
  }

  db.lastId, err = strconv.ParseInt(matches[2], 10, 64)
  return
}

func (db *Csv) writeFrontMatter(version int, lastId int64) (err error) {
  db.Mutex.Lock()
  defer db.Mutex.Unlock()

  var f *os.File
  f, err = os.OpenFile(db.Filename, os.O_WRONLY, 0644)
  if err != nil {
    return
  }
  defer f.Close()

  data := fmt.Sprintf("# version: %03d, last-id: %019d\n", version, lastId)
  _, err = f.Write([]byte(data))

  return
}

func (db *Csv) writeRecord(record []string) (err error) {
  db.Mutex.Lock()
  defer db.Mutex.Unlock()

  var f *os.File
  f, err = os.OpenFile(db.Filename, os.O_WRONLY | os.O_APPEND, 0644)
  if err != nil {
    return
  }
  defer f.Close()

  w := csv.NewWriter(f)
  err = w.Write(record)
  if err == nil {
    w.Flush()
  }
  return
}

func (db *Csv) Version() (version int, err error) {
  return db.version, nil
}

func (db *Csv) Migrate() (err error) {
  for db.version < CsvVersion {
    switch db.version {
    case 0:
      err = db.writeFrontMatter(1, 1)
      if err == nil {
        err = db.writeRecord([]string{"id", "name", "project", "tags",
          "start", "end"})
      }
    }
    if err != nil {
      return
    }
    db.version++
  }
  return
}

func (db *Csv) SaveActivity(activity *Activity) (err error) {
  /* FIXME: need mutex for id */
  id := db.lastId + 1
  record := make([]string, 6)
  record[0] = strconv.FormatInt(id, 10)
  record[1] = activity.Name
  record[2] = activity.Project
  record[3] = activity.TagList()
  record[4] = activity.Start.Format(time.RFC3339Nano)
  record[5] = activity.End.Format(time.RFC3339Nano)

  err = db.writeRecord(record)
  if err != nil {
    return
  }
  activity.Id = id
  db.lastId = id

  err = db.writeFrontMatter(db.version, db.lastId)
  return
}

func (db *Csv) FindActivity(id int64) (activity *Activity, err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.Open(db.Filename)
  if err != nil {
    return
  }
  defer f.Close()

  err = db.seekPastFrontMatter(f)
  if err != nil {
    return
  }

  r := csv.NewReader(f)

  var record []string
  n := 0
  for {
    record, err = r.Read()
    if err != nil {
      if err == io.EOF {
        err = nil
      }
      break
    }
    fmt.Println(record)
    if n > 0 {
      /* Ignore first row */
      var recordId int64
      recordId, err = strconv.ParseInt(record[0], 10, 64)
      if err != nil {
        return
      }
      fmt.Println(recordId, id)
      if recordId == id {
        activity = &Activity{Id: recordId, Name: record[1], Project: record[2]}
        activity.SetTagList(record[3])

        activity.Start, err = time.Parse(time.RFC3339Nano, record[4])
        if err != nil {
          return
        }
        activity.End, err = time.Parse(time.RFC3339Nano, record[5])
        if err != nil {
          return
        }
        break
      }
    }
    n += 1
  }

  return
}
