# rebuy-go-sdk

[![GoDoc](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk?status.svg)](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk) [![Build Status](https://travis-ci.org/rebuy-de/rebuy-go-sdk.svg?branch=master)](https://travis-ci.org/rebuy-de/rebuy-go-sdk)

Library for our Golang projects

> **Development Status** *rebuy-go-sdk* is designed for internal use. Since it
> uses [Semantic Versioning](https://semver.org/) it is safe to use, but expect
> big changes between major version updates.


## Major Release Notes

Note: `vN` is the new release (eg `v3`) and `vP` is the previous one (eg `v2`).

1. Create a new branch `release-vN` to avoid breaking changes getting into the previous release.
2. Do your breaking changes in the branch.
3. Update the imports everywhere:
   * `find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vO#github.com/rebuy-de/rebuy-go-sdk/vP#g' {} +`
4. Merge your branch.
5. Add Release on GitHub.
