---
id: M0006
title: Use webutil.NewServer
date: 2024-09-20
sdk_version: v8
type: minor
---

# Use webutil.NewServer

## Reasoning

The setup of the HTTP server has a lot of repetition in every project. Therefore, we move it into the SDK.
