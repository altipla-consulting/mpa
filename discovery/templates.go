package discovery

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"libs.altipla.consulting/env"
)

func TemplatesLocalAnchor(subfolder string) string {
	if env.IsLocal() {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			if _, err := os.Stat(filepath.Join(wd, "go.mod")); err != nil {
				if os.IsNotExist(err) {
					wd = filepath.Dir(wd)
					continue
				}

				log.Fatal(err)
			}

			return filepath.Join(wd, subfolder)
		}
	}
	return ""
}
