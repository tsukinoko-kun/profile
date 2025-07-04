package main

import (
	"os"
	"path/filepath"
)

func getDataStoreLocation() string {
	if xdgDataHome, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
		return filepath.Join(xdgDataHome, "profiler", "profiler.json")
	}
	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", "share", "profiler", "profiler.json")
}
