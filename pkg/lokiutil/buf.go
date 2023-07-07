package lokiutil

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/afiskon/promtail-client/logproto"
	"github.com/rebuy-de/rebuy-go-sdk/v6/pkg/cmdutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func makeBatch(buffer map[string][]string) Batch {
	result := []*logproto.Stream{}

	for labels, messages := range buffer {
		stream := logproto.Stream{
			Labels: labels,
		}

		for _, message := range messages {
			entry := logproto.Entry{
				// We need to use "now" and not the message timestamp, because
				// we need to guarantee message order.
				Timestamp: timestamppb.Now(),
				Line:      message,
			}
			stream.Entries = append(stream.Entries, &entry)
		}

		result = append(result, &stream)
	}

	return result
}

func splitLabels(m Message, hostname string, keys []string) (string, string) {
	labels := map[string]interface{}{
		"project": cmdutil.Name,
		"source":  hostname,
	}

	for _, k := range keys {
		value, ok := m[k]
		if ok {
			labels[k] = value
			delete(m, k)
		}
	}

	l := encodeLabels(labels)

	p, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return string(l), string(p)
}

// Loki uses some weird format for their labels. Therefore we have to marshal
// it by ourselves.
func encodeLabels(labels map[string]interface{}) string {
	parts := []string{}

	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%#v", k, v))
	}

	sort.Strings(parts)

	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}
