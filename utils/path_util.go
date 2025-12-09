package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func GetCurrentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	return cwd
}

const playwrightBaseDir = "/opt/playwright-browsers"

func DoesPlaywrightDirectoryExist() bool {
	_, err := os.Stat(playwrightBaseDir)
	return err == nil
}

// Find Chromium installed by Playwright inside playwrightBaseDir
func FindChromiumPath() (string, error) {
	entries, err := os.ReadDir(playwrightBaseDir)
	if err != nil {
		return "", fmt.Errorf("cannot read Playwright base dir: %w", err)
	}

	for _, entry := range entries {
		// Folder name must start with chromium-xxxx
		if strings.HasPrefix(entry.Name(), "chromium-") {
			candidate := filepath.Join(playwrightBaseDir, entry.Name(), "chrome-linux", "chrome")
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no Playwright chromium-* folder found in %s", playwrightBaseDir)
}
