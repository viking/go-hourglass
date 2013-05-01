package hourglass

import (
  "testing"
  "time"
)

func TestDefaultClock_Now(t *testing.T) {
  clock := DefaultClock{}
  for i := 0; i < 1000; i++ {
    duration := time.Since(clock.Now())
    if duration > time.Millisecond {
      t.Error("clock was off by", duration)
    }
  }
}

func TestDefaultClock_Since(t *testing.T) {
  clock := DefaultClock{}
  when := time.Date(2013, time.April, 29, 12, 0, 0, 0, time.Local)
  for i := 0; i < 1000; i++ {
    duration := clock.Since(when)
    realDuration := time.Since(when)
    if (realDuration - duration) > time.Millisecond {
      t.Error("expected '%s', got '%s'", realDuration, duration)
    }
  }
}

func TestDefaultClock_Local(t *testing.T) {
  clock := DefaultClock{}
  when := time.Now().UTC()
  if when.Local() != clock.Local(when) {
    t.Error("expected '%s', got '%s'", when.Local(), clock.Local(when))
  }
}
