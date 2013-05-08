package hourglass

import (
  "encoding/csv"
  "os"
  "io"
  "io/ioutil"
  "bufio"
  "regexp"
  "strconv"
  "sync"
  "errors"
  "fmt"
  "time"
  "bytes"
)

const CsvVersion = 1

var ErrBadFrontMatter = errors.New("invalid front matter")

type Csv struct {
  Filename string
  Mutex sync.RWMutex
  valid bool
  version int
  lastId int64
}

func NewCsv(filename string) (db *Csv, err error) {
  db = &Csv{Filename: filename}
  err = db.readFrontMatter()
  db.valid = err == nil
  return
}

func (db *Csv) Valid() (bool, error) {
  return db.valid, nil
}

func (db *Csv) seekToHeader(f *os.File) (pos int64, err error) {
  /* Front matter is 45 bytes long */
  pos, err = f.Seek(45, 0)
  return
}

func (db *Csv) seekToData(f *os.File) (pos int64, err error) {
  /* Header is: id,name,project,tags,start,end */
  pos, err = db.seekToHeader(f)
  if err != nil {
    return
  }

  pos, err = f.Seek(31, 1)
  return
}

func (db *Csv) readFrontMatter() (err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.OpenFile(db.Filename, os.O_RDONLY | os.O_CREATE, 0644)
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

func (db *Csv) appendRecord(record []string) (err error) {
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

func (db *Csv) writeBytes(pos int64, data []byte) (err error) {
  db.Mutex.Lock()
  defer db.Mutex.Unlock()

  var f *os.File
  f, err = os.OpenFile(db.Filename, os.O_WRONLY, 0644)
  if err != nil {
    return
  }
  defer f.Close()

  _, err = f.Seek(pos, 0)
  if err != nil {
    return
  }

  _, err = f.Write(data)
  return
}

func (db *Csv) readAll(pos int64) (data []byte, err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.Open(db.Filename)
  if err != nil {
    return
  }
  defer f.Close()

  _, err = f.Seek(pos, 0)
  if err != nil {
    return
  }

  data, err = ioutil.ReadAll(f)
  return
}

func (db *Csv) Version() (version int, err error) {
  return db.version, nil
}

func (db *Csv) Migrate() (err error) {
  for db.version < CsvVersion {
    switch db.version {
    case 0:
      err = db.writeFrontMatter(1, 0)
      if err == nil {
        err = db.appendRecord([]string{"id", "name", "project", "tags",
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

func (db *Csv) activityToRecord(activity *Activity) (record []string) {
  record = make([]string, 6)
  record[0] = strconv.FormatInt(activity.Id, 10)
  record[1] = activity.Name
  record[2] = activity.Project
  record[3] = activity.TagList()
  record[4] = activity.Start.Format(time.RFC3339Nano)
  record[5] = activity.End.Format(time.RFC3339Nano)
  return
}

func (db *Csv) recordToActivity(record []string) (activity *Activity, err error) {
  activity = new(Activity)
  activity.Id, err = strconv.ParseInt(record[0], 10, 64)
  if err != nil {
    return
  }

  activity.Name = record[1]
  activity.Project = record[2]
  activity.SetTagList(record[3])

  activity.Start, err = time.Parse(time.RFC3339Nano, record[4])
  if err != nil {
    return
  }
  activity.End, err = time.Parse(time.RFC3339Nano, record[5])
  return
}

func (db *Csv) createActivity(activity *Activity) (err error) {
  /* FIXME: need mutex for id */
  activity.Id = db.lastId + 1
  record := db.activityToRecord(activity)

  err = db.appendRecord(record)
  if err != nil {
    activity.Id = 0
    return
  }
  db.lastId = activity.Id

  err = db.writeFrontMatter(db.version, db.lastId)
  return
}

func (db *Csv) updateActivity(activity *Activity) (err error) {
  var pos int64
  var line []byte
  pos, line, err = db.findActivityLine(activity.Id)
  if err != nil {
    return
  }

  buf := new(bytes.Buffer)
  w := csv.NewWriter(buf)

  record := db.activityToRecord(activity)
  err = w.Write(record)
  if err != nil {
    return
  }
  w.Flush()

  var newLine []byte
  newLine, err = buf.ReadBytes('\n')
  if err != nil {
    return
  }

  /* If the resulting record is the same length, just overwrite it */
  if len(line) == len(newLine) {
    err = db.writeBytes(pos, newLine)
  } else {
    /* Save the data past the line, write the line, then put the data back */
    var data []byte
    data, err = db.readAll(pos + int64(len(line)))
    if err != nil {
      return
    }
    err = db.writeBytes(pos, newLine)
    if err != nil {
      return
    }
    err = db.writeBytes(pos + int64(len(newLine)), data)
  }

  return
}

func (db *Csv) SaveActivity(activity *Activity) (err error) {
  if activity.Id > 0 {
    err = db.updateActivity(activity)
  } else {
    err = db.createActivity(activity)
  }
  return
}

func (db *Csv) findActivityLine(id int64) (pos int64, line []byte, err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.Open(db.Filename)
  if err != nil {
    return
  }
  defer f.Close()

  pos, err = db.seekToData(f)
  if err != nil {
    return
  }

  r := bufio.NewReader(f)

  for {
    line, err = r.ReadBytes(',')
    if err != nil {
      break
    }

    var recordId int64
    recordId, err = strconv.ParseInt(string(line[:len(line)-1]), 10, 64)
    if err != nil {
      /* TODO: be more fault tolerant */
      break
    }

    var rest []byte
    rest, err = r.ReadBytes('\n')
    if err != nil {
      /* TODO: be more fault tolerant */
      break
    }

    if recordId == id {
      line = append(line, rest...)
      break
    }
    pos += int64(len(line)) + int64(len(rest))
  }

  return
}

func (db *Csv) FindActivity(id int64) (activity *Activity, err error) {
  var line []byte

  _, line, err = db.findActivityLine(id)
  if err == io.EOF {
    err = ErrNotFound
    return
  }

  buf := bytes.NewBuffer(line)
  r := csv.NewReader(buf)

  var record []string
  record, err = r.Read()
  if err != nil {
    return
  }
  activity, err = db.recordToActivity(record)
  return
}

func (db *Csv) findActivities(filter func(*Activity) bool) (activities []*Activity, err error) {
  db.Mutex.RLock()
  defer db.Mutex.RUnlock()

  var f *os.File
  f, err = os.Open(db.Filename)
  if err != nil {
    return
  }
  defer f.Close()

  _, err = db.seekToData(f)
  if err != nil {
    return
  }

  r := csv.NewReader(f)

  var records [][]string
  records, err = r.ReadAll()
  if err != nil {
    return
  }

  activities = make([]*Activity, 0, len(records))
  for _, record := range records {
    var activity *Activity
    activity, err = db.recordToActivity(record)
    if err != nil {
      return
    }
    if filter == nil || filter(activity) {
      activities = append(activities, activity)
    }
  }
  return
}

func (db *Csv) FindAllActivities() (activities []*Activity, err error) {
  activities, err = db.findActivities(nil)
  return
}

func (db *Csv) FindRunningActivities() (activities []*Activity, err error) {
  filter := func(a *Activity) bool { return a.IsRunning() }
  activities, err = db.findActivities(filter)
  return
}

func (db *Csv) FindActivitiesBetween(lower time.Time, upper time.Time) (activities []*Activity, err error) {
  filter := func(a *Activity) bool {
    return (a.Start.Equal(lower) || a.Start.After(lower)) && a.Start.Before(upper)
  }
  activities, err = db.findActivities(filter)
  return
}
