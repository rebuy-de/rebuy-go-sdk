package webutil

type CDNMirrorSource struct {
	URL    string
	Target string
	Minify CDNMirrorMinifier
}

type CDNMirrorMinifier string

const (
	CDNMirrorMinifyJS  = "js"
	CDNMirrorMinifyCSS = "css"
)

func CDNMirrorSourceHotwiredTurbo() CDNMirrorSource {
	return CDNMirrorSource{
		URL:    "https://unpkg.com/@hotwired/turbo@7.1.0/dist/turbo.es2017-umd.js",
		Target: "hotwired-turbo-7.1.0-min.js",
		Minify: CDNMirrorMinifyJS,
	}
}

func CDNMirrorSourceBootstrap() CDNMirrorSource {
	return CDNMirrorSource{
		URL:    "https://unpkg.com/bootstrap@5.1.3/dist/css/bootstrap.min.css",
		Target: "bootstrap-5.1.3-min.js",
	}
}
