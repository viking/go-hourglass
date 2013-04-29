package clock

import (
  "testing"
  "time"
)

func TestRealClock_Now(t *testing.T) {
  clock := RealClock{}
  for i := 0; i < 1000; i++ {
    duration := time.Since(clock.Now())
    if duration > time.Microsecond {
      t.Error("clock was off by", duration)
    }
  }
}

func TestRealClock_Local(t *testing.T) {
  clock := RealClock{}
  when := time.Now().UTC()
  if when.Local() != clock.Local(when) {
    t.Error("expected '%s', got '%s'", when.Local(), clock.Local(when))
  }
}
