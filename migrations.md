# Migrations

This file contains a list of tasks that are either required or at least
strongly recommended to align projects using this SDK.

## 2024-06-14 Remove all uses of `github.com/pkg/errors`

### Reasoning

[github.com/pkg/errors](https://github.com/pkg/errors)` is deprecated. Since the built-in `errors` package improved a bit in the recent Go versions, we should remove all uses of `github.com/pkg/errors` and replace it with the `errors` package.

### Hints

* Use the pattern `return fmt.Errorf("something happenend with %#v: %w", someID, err)`
* The stack trace feature gets lost. Therefore it is suggested to properly add error messages each time handling errors.


## 2024-07-05 Switch to Jet Templates

### Reasoning

The builtin template engine has some shortcommings like being able to pass multiple and optional parameters to a template fragment. For example this block definition is not possible to builtin template engine:

```
{{ block entityLink(label, id, tab, extraClasses) }}
  {{ e := findEntity(.) }}
  <a class="ry-nowrap ry-underline {{ extraClasses }}" href="/{{.}}{{if id}}/{{id}}{{end}}{{ if tab }}/{{tab}}{{end}}">
    <span class="icon">
      <i class="fa-solid {{ e.IconClasses }}"></i>
    </span>
    {{ if label }}
    <span>{{ label }}</span>
    {{ else if id }}
    <span>{{ id }}</span>
    {{ else }}
    <span>{{ e.Plural }}</span>
    {{ end }}
  </a>
{{ end }}
```

### Hints

* The handler function signatures changed from `func(*webutil.View, *http.Request) webutil.Response` to `func(*http.Request) webutil.Response`.
* Wrap function changes from `webutil.ViewHandler.Wrap` to `webutil.WrapView`.
* Since the `View` interface is not part of the function signature, it needs to be added to the struct where the handler function is attached to.
* Non-HTML responses are now created with functions directly from the `webutil` package. For example `return webutilext.ViewError(http.StatusInternalServerError, err)` instead of `return v.Error(http.StatusBadRequest, fmt.Errorf(`unknown value for "until"`))`. where `v` was passed to the handler.

### Examples

Jet Set for Development:

```go
jet.NewSet(
	jet.NewOSFileSystemLoader("./pkg/app/web/templates"),
	jet.InDevelopmentMode(),
)
```

Jet Set for Production:

```go
jet.NewSet(
	webutil.JetFSLoader{FS: templateFS},
)
```

Jet Viewer based on Jet Set:

```go
webutilext.NewJetViewer(
	js,
	webutil.JetVarOption("clusterName", cn),
	webutil.JetVarOption("assetPath", "/assets/"+prefix),
)
```
