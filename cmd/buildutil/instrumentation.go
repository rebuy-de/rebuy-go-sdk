package main

import (
	"encoding/json"
	"os"
	"path"
	"sync"
	"time"

	"github.com/rebuy-de/rebuy-go-sdk/v7/cmd/buildutil/internal/typeutil"
	"github.com/sirupsen/logrus"
)

type Instrumentation struct {
	l *sync.Mutex

	Durations struct {
		Steps     *DurationMap `json:"steps,omitempty"`
		Testing   *DurationMap `json:"test,omitempty"`
		Building  *DurationMap `json:"build,omitempty"`
		Artifacts *DurationMap `json:"artifacts,omitempty"`
		Upload    *DurationMap `json:"upload,omitempty"`
	} `json:",omitempty"`

	Sizes map[string]typeutil.JSONBytes `json:",omitempty"`
}

func NewInstrumentation() *Instrumentation {
	inst := new(Instrumentation)
	inst.l = new(sync.Mutex)

	inst.Durations.Steps = NewDurationMap()
	inst.Durations.Testing = NewDurationMap()
	inst.Durations.Building = NewDurationMap()
	inst.Durations.Artifacts = NewDurationMap()
	inst.Durations.Upload = NewDurationMap()

	return inst
}

func (i *Instrumentation) ReadSize(name string) {
	fi, err := os.Stat(path.Join("dist", name))
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get size of %s", name)
		return
	}

	i.l.Lock()
	defer i.l.Unlock()

	if i.Sizes == nil {
		i.Sizes = map[string]typeutil.JSONBytes{}
	}

	i.Sizes[name] = typeutil.JSONBytes{
		Size: fi.Size(),
	}
}

func Stopwatch(target *typeutil.JSONDuration) func() {
	start := time.Now()
	return func() {
		target.Duration = time.Since(start).Truncate(time.Millisecond)
	}
}

type DurationMap struct {
	m map[string]typeutil.JSONDuration
	l *sync.Mutex
}

func NewDurationMap() *DurationMap {
	return &DurationMap{
		m: map[string]typeutil.JSONDuration{},
		l: new(sync.Mutex),
	}
}

func (m *DurationMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.m)
}

func (m *DurationMap) Stopwatch(name string) func() {
	start := time.Now()

	return func() {
		d := time.Since(start).Truncate(time.Millisecond)

		m.l.Lock()
		defer m.l.Unlock()

		m.m[name] = typeutil.JSONDuration{
			Duration: d,
		}
	}
}
