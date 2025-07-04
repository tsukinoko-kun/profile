package main

import (
	"os"
	"path/filepath"
)

func getDataStoreLocation() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, "Library", "Application Support", "dev.frankmayer", "profiler", "data.json")
}
