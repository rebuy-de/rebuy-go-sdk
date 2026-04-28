---
id: M0004
title: Isolate HTTP handlers
date: 2024-09-13
sdk_version: v8
type: minor
---

# Isolate HTTP handlers

## Reasoning

Our previous approach to have a single `Server` struct and to put all logic there gets messy pretty fast. Worse than
that it is hard to refactor this to separate this into multiple structs or packages. Dependency injection will help us
to separate those things in the very beginning of a project without manual wiring of dependencies. One step towards this
to split up all HTTP handlers into separate files.

## Hints

* The handlers should be moved into the `pkg/app/handlers` package.
* The handler struct should have a `New...` constructor with all required dependencies as parameters.
* The handler struct should implement `interface { Register(chi.Router) }`, which gets called once to set up the routes.
  This will later be used for dependency injection integration.

## Example

```go
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/platform-inventory/pkg/bll/webutilext"
	"github.com/rebuy-de/platform-inventory/pkg/dal/sqlc"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

type KubeEventHandler struct {
	sqlc   *sqlc.Queries
	viewer *webutilext.JetViewer
}

func NewKubeEventHandler(
    sqlc *sqlc.Queries,
    viewer *webutilext.JetViewer,
) *KubeEventHandler {
	return &KubeEventHandler{
		sqlc:   sqlc,
		viewer: viewer,
	}
}

func (h *KubeEventHandler) Register(router chi.Router) {
	router.Get("/kube/events", webutilext.WrapView(h.list))
	router.Get("/kube/events/table-fragment", webutilext.WrapView(h.listFragment))
}

func (h *KubeEventHandler) list(r *http.Request) webutil.Response {
	events, err := h.sqlc.ListKubeEvents(r.Context())
	if err != nil {
		return webutilext.ViewError(http.StatusInternalServerError, err)
	}

	return h.viewer.HTML(http.StatusOK, "kube_event_list.html", events)
}

func (h *KubeEventHandler) listFragment(r *http.Request) webutil.Response {
	events, err := h.sqlc.ListKubeEvents(r.Context())
	if err != nil {
		return webutilext.ViewError(http.StatusInternalServerError, err)
	}

	return h.viewer.HTML(http.StatusOK, "frames/kube_event_table.html", events)
}
```
