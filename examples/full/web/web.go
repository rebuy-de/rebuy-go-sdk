package web

import (
	"embed"
	"io/fs"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

//go:generate yarn install
//go:generate yarn build

//go:embed all:dist/*
var embedded embed.FS

func DevFS() webutil.AssetFS {
	return os.DirFS("web/dist")
}

func ProdFS() webutil.AssetFS {
	result, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}

	return result
}
