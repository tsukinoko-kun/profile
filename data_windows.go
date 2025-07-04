package main

import (
	"os"
	"path/filepath"
)

func getDataStoreLocation() string {
	appdata := os.Getenv("APPDATA")
	return filepath.Join(appdata, "dev.frankmayer", "profiler.json")
}
