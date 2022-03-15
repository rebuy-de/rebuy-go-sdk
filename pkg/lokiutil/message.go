package lokiutil

import "github.com/afiskon/promtail-client/logproto"

type Message = map[string]interface{}

type Batch = []*logproto.Stream
