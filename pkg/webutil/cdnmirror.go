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
		Target: "bootstrap-5.1.3-min.css",
	}
}

func CDNMirrorSourceFontAwesomeSprites() CDNMirrorSource {
	return CDNMirrorSource{
		URL:    "https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.1.2/sprites/solid.svg",
		Target: "font-awesome-6.1.2-sprites-solid.svg",
	}
}

func CDNMirrorSourceBulma() CDNMirrorSource {
	return CDNMirrorSource{
		URL:    "https://cdn.jsdelivr.net/npm/bulma@0.9.4/css/bulma.min.css",
		Target: "bulma-0.7.4.min.css",
	}
}
