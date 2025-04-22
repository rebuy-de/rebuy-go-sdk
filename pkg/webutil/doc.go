// Package webutil provides functions for making developing a website easier.
//
// # HTTP Handlers with webutil
//
// The SDK provides a streamlined approach to HTTP request handling across projects.
//
// ## Creating Handlers
//
// Handlers should be organized as structs that follow the SDK's handler registration pattern:
//
//	// Define a handler struct with dependencies
//	type MyHandler struct {
//	    // Dependencies injected via constructor
//	    store     *SomeStore
//	    viewer    *webutil.GoTemplateViewer
//	}
//
//	// Constructor that creates a new handler instance
//	func NewMyHandler(store *SomeStore, viewer *webutil.GoTemplateViewer) *MyHandler {
//	    return &MyHandler{
//	        store:  store,
//	        viewer: viewer,
//	    }
//	}
//
//	// Register routes on a chi Router
//	func (h *MyHandler) Register(router chi.Router) {
//	    router.Get("/api/resource", webutil.WrapView(h.handleGetResource))
//	    router.Post("/api/resource", webutil.WrapView(h.handleCreateResource))
//	}
//
//	// Handler functions return a webutil.Response (which is an http.HandlerFunc)
//	func (h *MyHandler) handleGetResource(r *http.Request) webutil.Response {
//	    // Get data from store
//	    data, err := h.store.Get(r.Context(), "some-id")
//	    if err != nil {
//	        return webutil.ViewError(http.StatusInternalServerError, err)
//	    }
//
//	    // Return appropriate response
//	    return h.viewer.HTML(http.StatusOK, "resource.html", data)
//	    // Or for API endpoints
//	    // return webutil.ViewJSON(http.StatusOK, data)
//	}
//
// ## Response Helpers
//
// The webutil package provides several helper functions to generate HTTP responses:
//
//	// HTML response using a template
//	return h.viewer.HTML(http.StatusOK, "template.html", data)
//
//	// JSON response
//	return webutil.ViewJSON(http.StatusOK, data)
//
//	// Error response
//	return webutil.ViewError(http.StatusInternalServerError, err)
//
//	// Formatted error
//	return webutil.ViewErrorf(http.StatusBadRequest, "invalid parameter: %s", param)
//
//	// Redirect response
//	return webutil.ViewRedirect(http.StatusSeeOther, "/new-location")
//
//	// Empty response
//	return webutil.ViewNoContent(http.StatusNoContent)
//
//	// Inline HTML (for HTMX partial updates)
//	return webutil.ViewInlineHTML(http.StatusOK, "<span>Updated %s</span>", item)
//
// ## Template Viewers
//
// The SDK supports different template engines:
//
// 1. GoTemplateViewer - For standard Go HTML templates
//
//	viewer := webutil.NewGoTemplateViewer(templateFS,
//	    webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
//	    webutil.SimpleTemplateFuncMaps(template.FuncMap{
//	        "truncate": TruncateFunction,
//	        "format": FormatFunction,
//	    }),
//	)
//
// 2. JetViewer - For the Jet template engine (provided by extension packages)
//
//	// Create a Jet loader from an fs.FS
//	loader := webutilext.JetFSLoader{FS: templateFS}
//	jetSet := jet.NewSet(loader)
//
//	// Create the viewer with functions
//	viewer := webutilext.NewJetViewer(jetSet,
//	    webutilext.JetFunctionOption("formatTime", FormatTimeFunction),
//	    webutilext.JetFunctionMapOption(map[string]any{
//	        "truncate": TruncateFunction,
//	    }),
//	)
//
// 3. Templ - For the templ type-safe HTML template engine
//
// Templ can be integrated with the SDK's webutil framework by creating a custom viewer type
// that adapts templ components to return webutil.Response functions.
//
//	// suggested content for pkg/app/templates/view.go
//	package templates
//
//	import (
//	    "fmt"
//	    "net/http"
//
//	    "github.com/a-h/templ"
//	    "github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
//	    "github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
//	)
//
//	//go:generate go run github.com/a-h/templ/cmd/templ generate
//	//go:generate go run github.com/a-h/templ/cmd/templ fmt .
//
//	type Viewer struct {
//	    assetPathPrefix webutil.AssetPathPrefix
//	}
//
//	func New(
//	    assetPathPrefix webutil.AssetPathPrefix,
//	) *Viewer {
//	    return &Viewer{
//	        assetPathPrefix: assetPathPrefix,
//	    }
//	}
//
//	func (v *Viewer) assetPath(path string) string {
//	    return fmt.Sprintf("/assets/%v%v", v.assetPathPrefix, path)
//	}
//
//	func View(status int, node templ.Component) webutil.Response {
//	    return func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("Content-Type", "text/html; charset=utf-8")
//	        w.WriteHeader(status)
//
//	        err := node.Render(r.Context(), w)
//	        if err != nil {
//	            logutil.Get(r.Context()).Error(err)
//	        }
//	    }
//	}
//
// Usage with webutil handler:
//
//	func (h *Handler) handleHome(r *http.Request) webutil.Response {
//	    data := GetPageData()
//	    return templates.View(http.StatusOK, HomePage(data))
//	}
//
// ## Handler Registration with Dependency Injection
//
// The SDK uses the dig dependency injection container to manage and register HTTP handlers:
//
//	func RunServer(ctx context.Context, c *dig.Container) error {
//	    // Register the template viewer
//	    err := c.Provide(func(templateFS fs.FS, assetPathPrefix webutil.AssetPathPrefix) *webutil.GoTemplateViewer {
//	        return webutil.NewGoTemplateViewer(templateFS,
//	            webutil.AuthTemplateFunctions,
//	            webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
//	            // Add more template functions as needed
//	        )
//	    })
//	    if err != nil {
//	        return err
//	    }
//
//	    // Register handlers with the dig container
//	    err = errors.Join(
//	        // Each of these calls registers a handler that implements Handler interface
//	        webutil.ProvideHandler(c, handlers.NewUserHandler),
//	        webutil.ProvideHandler(c, handlers.NewResourceHandler),
//	        webutil.ProvideHandler(c, handlers.NewWebhookHandler),
//
//	        // Register server last
//	        runutil.ProvideWorker(c, webutil.NewServer),
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    // Run all registered workers (including the web server)
//	    return runutil.RunProvidedWorkers(ctx, c)
//	}
//
// ## Server Setup
//
// The SDK provides a webutil.Server type that manages HTTP server setup:
//
//	// In your main Runner.Run method:
//	func (r *Runner) Run(ctx context.Context) error {
//	    // Set up the dig container
//	    c := dig.New()
//
//	    // Provide dependencies (databases, clients, configs)
//	    err := errors.Join(
//	        c.Provide(func() fs.FS { return templateFS }),
//	        c.Provide(webutil.AssetDefaultProd), // Provides AssetPathPrefix and AssetCacheDuration
//	        // ... other dependencies
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    // Run the server
//	    return RunServer(ctx, c)
//	}
package webutil
