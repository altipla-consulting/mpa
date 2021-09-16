package vite

import (
	"encoding/json"
	"os"

	"libs.altipla.consulting/errors"
)

type Manifest map[string]Chunk

type Chunk struct {
	File string
	CSS  []string
}

func ReadManifest() (Manifest, error) {
	f, err := os.Open("dist/manifest.json")
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, nil
		}
		return nil, errors.Trace(err)
	}
	defer f.Close()

	manifest := make(Manifest)
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, errors.Trace(err)
	}

	return manifest, nil
}
