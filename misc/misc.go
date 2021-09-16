package misc

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"libs.altipla.consulting/cloudrun"
	"libs.altipla.consulting/env"
	"libs.altipla.consulting/routing"
)

func Register(r *cloudrun.WebServer, baseTemplate string) {
	go func() {
		// Touch template to reload the page every time we change the Go implementation.
		_ = os.Chtimes(baseTemplate, time.Now(), time.Now())
	}()

	r.Get("/robots.txt", fileHandler("robots.txt"))
	r.Get("/favicon.ico", fileHandler("favicon.ico"))
	r.Get("/apple-touch-icon.png", fileHandler("apple-touch-icon.png"))
}

func fileHandler(path string) routing.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if env.IsLocal() {
			http.ServeFile(w, r, filepath.Join("..", "public", path))
		} else {
			http.ServeFile(w, r, filepath.Join("public", path))
		}
		return nil
	}
}