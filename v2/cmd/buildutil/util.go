// +build linux darwin
package main

import (
	"path"
	"strings"
)

func RelativePath(basepath, targpath string) string {
	basepath = path.Clean(basepath)
	targpath = path.Clean(targpath)

	if basepath == targpath {
		return "."
	}

	result := strings.TrimPrefix(targpath, basepath+"/")
	return path.Clean(result)
}
