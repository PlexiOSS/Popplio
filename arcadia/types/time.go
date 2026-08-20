package types

import (
	"encoding/json"
	"time"
)

type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t}
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format())
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}

	t.Time = parsed
	return nil
}

func (t Timestamp) Format() string {
	utc := t.Time.UTC()
	nanos := utc.Nanosecond()

	var layout string
	switch {
	case nanos == 0:
		layout = "2006-01-02T15:04:05"
	case nanos%1_000_000 == 0:
		layout = "2006-01-02T15:04:05.000"
	case nanos%1_000 == 0:
		layout = "2006-01-02T15:04:05.000000"
	default:
		layout = "2006-01-02T15:04:05.000000000"
	}

	return utc.Format(layout) + "Z"
}

func (t Timestamp) String() string {
	return t.Format()
}

type OptTimestamp = *Timestamp

func TimestampPtr(t *time.Time) *Timestamp {
	if t == nil {
		return nil
	}

	ts := NewTimestamp(*t)
	return &ts
}
