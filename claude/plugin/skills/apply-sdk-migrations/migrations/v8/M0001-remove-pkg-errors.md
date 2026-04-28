---
id: M0001
title: Remove all uses of github.com/pkg/errors
date: 2024-06-14
sdk_version: v8
type: minor
---

# Remove all uses of `github.com/pkg/errors`

## Reasoning

[github.com/pkg/errors](https://github.com/pkg/errors) is deprecated. Remove it and use `github.com/pkg/errors` instead.

## Hints

* Use the pattern `return fmt.Errorf("something happenend with %#v: %w", someID, err)`
* The stack trace feature gets lost. Therefore it is suggested to properly add error messages each time handling errors.
