package lokiutil

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/afiskon/promtail-client/logproto"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
)

type Client struct {
	url  string
	http *http.Client

	maxSize int
	maxWait time.Duration
	keys    []string

	messages chan Message
	errc     chan error
	stopped  chan struct{}
}

func New(url string, maxSize int, maxWait time.Duration, keys []string) *Client {
	c := &Client{
		url:     url,
		http:    new(http.Client),
		maxWait: maxWait,
		maxSize: maxSize,
		keys:    keys,

		messages: make(chan Message, maxSize*2),
		errc:     make(chan error, 100),
		stopped:  make(chan struct{}),
	}

	return c
}

func (c *Client) Log(m Message) {
	select {
	case c.messages <- m:
	default:
	}
}

func (c *Client) Errc() <-chan error {
	return c.errc
}

func (c *Client) err(err error) {
	if err == nil {
		return
	}

	select {
	case c.errc <- err:
	default:
	}
}

// Stop sends the remaining messages and stops processing new ones.
func (c *Client) Stop() {
	close(c.messages)
	close(c.errc)
}

// Run processes the messages in the background. It needs to get stopped by
// Stop() to not lose messages. It is not controlled by a context, because
// logging should be the last component that gets stopped.
func (c *Client) Run() error {
	var (
		buffer = map[string][]string{}
		size   = 0
		done   = false
		timer  = time.NewTimer(c.maxWait)
	)

	defer close(c.stopped)

	hostname, err := os.Hostname()
	if err != nil {
		return errors.WithStack(err)
	}

	for !done {
		send := false
		select {
		case message, ok := <-c.messages:
			if !ok {
				// Channel is closed
				done = true
				break
			}

			l, p := splitLabels(message, hostname, c.keys)
			messages := buffer[l]
			messages = append(messages, p)
			buffer[l] = messages

			size++
			if size >= c.maxSize {
				send = true
			}
		case <-timer.C:
			send = true
		}

		if send {
			timer.Stop()
			select {
			case <-timer.C:
			default:
			}

			if size > 0 {
				batch := makeBatch(buffer)
				err := c.sendBatch(batch)
				c.err(err)
			}

			size = 0
			buffer = map[string][]string{}
			timer.Reset(c.maxWait)
		}
	}

	return nil
}

func (c *Client) sendBatch(batch Batch) error {
	buf, err := proto.Marshal(&logproto.PushRequest{
		Streams: batch,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	buf = snappy.Encode(nil, buf)

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(buf))
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := c.http.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != 204 {
		return errors.Errorf("unexpected HTTP status code %d: %s", resp.StatusCode, string(resBody))
	}

	return nil
}
