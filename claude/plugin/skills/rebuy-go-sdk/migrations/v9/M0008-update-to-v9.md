---
id: M0008
title: Update to v9
date: 2024-04-22
sdk_version: v9
type: major
---

# Update to v9

Check release notes and upgrade to v9.

* Many deprecated function got removed.
* Removed `lokiutil` (not used anywhere)
* Removed `podtuil` (use testcontainers instead)
* Removed `redisutil.Broadcast` (not used anywhere)
* `webutil.ViewRedirect` does not support format args anymore. Use `webutil.ViewRedirectf` instead.
* `webutil.AdminAPIListenAndServe` now uses options. Check go docs for new usage.
* Removed `cdnmirror`. Use Yarn or similar instead.
* Change signature of `cmdutil.Runner.Run` from `(context.Context)` to `(context.Context, []string)` in order to be able
  to access positional args.
