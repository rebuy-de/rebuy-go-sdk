package webutil

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/redisutil"
	"github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
)

type Renderer interface {
	Render(filename string, r *http.Request, d interface{}) (*bytes.Buffer, error)
}

type HotwiredBroadcast[T any] struct {
	broadcast *redisutil.Broadcast[HotwiredFrame[T]]
	renderer  Renderer
}

func NewHotwiredBroadcast[T any](client redisutil.BroadcastRediser, key string, renderer Renderer) (*HotwiredBroadcast[T], error) {
	redisBroadcast, err := redisutil.NewBroadcast[HotwiredFrame[T]](client, key)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &HotwiredBroadcast[T]{
		broadcast: redisBroadcast,
		renderer:  renderer,
	}, nil
}

type HotwiredFrame[T any] struct {
	Filename string
	Value    *T
}

func (b *HotwiredBroadcast[T]) AddFrame(ctx context.Context, filename string, value *T) error {
	err := b.broadcast.Add(ctx, &HotwiredFrame[T]{
		Filename: filename,
		Value:    value,
	})
	return errors.WithStack(err)
}

func (b *HotwiredBroadcast[T]) Handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.WithField("path", r.RequestURI).Debugf("accepting new connection")

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Errorf("accepting websocket connection failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close(websocket.StatusInternalError, "fail")

	id := "0-0"

	for r.Context().Err() == nil {
		v, newID, err := b.broadcast.Read(r.Context(), id)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			logrus.WithError(err).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		id = newID

		html, err := b.renderer.Render(v.Filename, r, v.Value)
		if err != nil {
			logrus.WithError(err).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.Write(r.Context(), websocket.MessageText, html.Bytes())
		if err != nil {
			logrus.WithError(err).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	c.Close(websocket.StatusNormalClosure, "")
}

func HotwiredTemplateFunctions(_ *http.Request) template.FuncMap {
	return template.FuncMap{
		"hotwiredImport": HotwiredImportTemplateFunction,
		"hotwiredStream": HotwiredStreamTemplateFunction,
	}
}

func HotwiredImportTemplateFunction() template.HTML {
	return template.HTML(`
      <script type="module">
        import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';
      </script>
      `)
}

func HotwiredStreamTemplateFunction(path string) template.HTML {
	path = "/" + strings.TrimLeft(path, "/")

	return template.HTML(fmt.Sprintf(`
      <script type="text/javascript">
        window.onload = function () {
          var l = window.location;
          var p = (l.protocol === "https:") ? "wss://" : "ws://";
          Turbo.connectStreamSource(new WebSocket(p + l.host + "%s"));
        };
      </script>
      `, path))
}
