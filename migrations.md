# Migrations

This file contains a list of tasks that are either required or at least
strongly recommended to align projects using this SDK.

## 2024-06-14 Remove all uses of `github.com/pkg/errors`

### Reasoning

[github.com/pkg/errors](https://github.com/pkg/errors)` is deprecated. Since the built-in `errors` package improved a bit in the recent Go versions, we should remove all uses of `github.com/pkg/errors` and replace it with the `errors` package.

### Hints

* Use the pattern `return fmt.Errorf("something happenend with %#v: %w", someID, err)`
* The stack trace feature gets lost. Therefore it is suggested to properly add error messages each time handling errors.
