package typeutil

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type JSONDuration struct {
	time.Duration
}

func (j JSONDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Duration.String())
}

type JSONBytes struct {
	Size int64
}

func (j JSONBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.String())
}

func (j JSONBytes) String() string {
	var (
		pre = []string{"B", "KiB", "MiB", "GiB", "TiB"}

		bits = math.Log2(float64(j.Size))
		pos  = math.Floor(bits / 10)
		exp  = math.Pow(1024, float64(pos))

		short = float64(j.Size) / exp
		sufix = pre[int(pos)]
	)

	if pos == 0 {
		return fmt.Sprintf("%.0f%s", short, sufix)
	}

	if short < 10 {
		return fmt.Sprintf("%.3f%s", short, sufix)
	}

	if short < 100 {
		return fmt.Sprintf("%.2f%s", short, sufix)
	}

	return fmt.Sprintf("%.1f%s", short, sufix)

}
