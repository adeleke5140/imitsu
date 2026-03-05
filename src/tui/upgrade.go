package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
)

const repo = "adeleke5140/imitsu"

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func latestVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func upgrade() error {
	fmt.Println("Checking for updates...")

	tag, err := latestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := tag
	if len(latest) > 0 && latest[0] == 'v' {
		latest = latest[1:]
	}

	if version != "" && latest == version {
		fmt.Printf("Already up to date (v%s)\n", version)
		return nil
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/itui-%s-%s", repo, tag, goos, goarch)

	fmt.Printf("Downloading itui v%s for %s/%s...\n", latest, goos, goarch)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d (no release for %s/%s?)", resp.StatusCode, goos, goarch)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "itui-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("download failed: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Find current binary path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine binary path: %w", err)
	}

	// Replace current binary
	if err := os.Rename(tmpFile.Name(), execPath); err != nil {
		// Rename may fail across filesystems, try copy
		return copyFile(tmpFile.Name(), execPath)
	}

	fmt.Printf("Upgraded to itui v%s\n", latest)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to write (try with sudo): %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
