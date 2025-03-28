package templates

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

//go:generate go run github.com/a-h/templ/cmd/templ generate
//go:generate go run github.com/a-h/templ/cmd/templ fmt .

type Viewer struct {
	assetPathPrefix webutil.AssetPathPrefix
}

func New(
	assetPathPrefix webutil.AssetPathPrefix,
) *Viewer {
	return &Viewer{
		assetPathPrefix: assetPathPrefix,
	}
}

func (v *Viewer) assetPath(path string) string {
	return fmt.Sprintf("/assets/%v%v", v.assetPathPrefix, path)
}

func View(status int, node templ.Component) webutil.Response {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)

		err := node.Render(r.Context(), w)
		if err != nil {
			logutil.Get(r.Context()).Error(err)
		}
	}
}