---
id: M0003
title: Change viewer interfaces of webutil
date: 2024-08-16
sdk_version: v8
type: minor
---

# Change viewer interfaces of webutil

## Reasoning

The previous interface is a bit awkward to use with dependency injection together with splitting the HTTP handlers into
multiple structs. Additionally there were cases where we wanted to use the `webutil.Response` type for convenience, but
did not actually need any HTML rendering.

Therefore the interfaces are changed this way:
* The new `webuitil.WrapView` function replaces the old `webuitl.ViewHandler.Wrap` function and does not require any
  template definitions.
* All `webutil.Response` functions, that do not need templates, are pure functions now (ie not attached to a type).
* The handler interface gets reduced to `func(*http.Request) Response`, so it does not contain the view parameter
  anymore. When using HTML, it is required to attach the `webutil.GoTemplateViewer` to the struct that implements the
  handler.
