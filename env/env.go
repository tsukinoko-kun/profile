package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	GOOGLE_API_KEY string
)

func init() {
	loadEnv()

	if googleApiKey, ok := os.LookupEnv("GOOGLE_API_KEY"); ok {
		GOOGLE_API_KEY = googleApiKey
	} else {
		fmt.Fprintln(os.Stderr, "GOOGLE_API_KEY is not set")
		os.Exit(1)
		return
	}
}

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	// read line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.Split(line, "#")[0]
		parts := strings.Split(line, "=")
		key := strings.TrimSpace(parts[0])
		if len(parts) < 2 {
			os.Setenv(key, "")
			continue
		}

		var value string
		value = strings.TrimSpace(strings.Join(parts[1:], " "))
		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("error reading .env file: %v\n", err)
		return
	}
}
