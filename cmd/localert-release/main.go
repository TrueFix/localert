// localert-release creates a self-update manifest for the platform binaries in bin/.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
	"localert/internal/updater"
)

func main() {
	version := flag.String("version", "", "semantic release version")
	dir := flag.String("dir", "bin", "directory containing release binaries")
	base := flag.String("base-url", "", "public directory URL (optional for a static feed; required for GitHub release assets)")
	notes := flag.String("notes", "A fresh little Localert release.", "release notes")
	flag.Parse()
	if err := generate(*version, *dir, *base, *notes); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(version, directory, base, notes string) error {
	version = strings.TrimPrefix(version, "v")
	if !semver.IsValid("v" + version) {
		return fmt.Errorf("invalid release version %q", version)
	}
	if base != "" {
		if err := updater.ValidateURL(base); err != nil {
			return err
		}
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
	}
	manifest := updater.Manifest{Version: version, Notes: notes, PublishedAt: time.Now().UTC().Format(time.RFC3339), Assets: map[string]updater.Asset{}}
	files := []struct{ platform, name string }{{"linux-amd64", "localert-linux-amd64"}, {"linux-arm64", "localert-linux-arm64"}, {"windows-amd64", "localert-windows-amd64.exe"}, {"darwin-amd64", "localert-macos-amd64"}, {"darwin-arm64", "localert-macos-arm64"}}
	var checksums strings.Builder
	for _, item := range files {
		file, err := os.Open(filepath.Join(directory, item.name))
		if err != nil {
			return err
		}
		hash := sha256.New()
		size, err := io.Copy(hash, file)
		file.Close()
		if err != nil {
			return err
		}
		address := item.name
		if base != "" {
			root, _ := url.Parse(base)
			part, _ := url.Parse(item.name)
			address = root.ResolveReference(part).String()
		}
		checksum := hex.EncodeToString(hash.Sum(nil))
		manifest.Assets[item.platform] = updater.Asset{URL: address, SHA256: checksum, Size: size}
		fmt.Fprintf(&checksums, "%s  %s\n", checksum, item.name)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "update-manifest.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(directory, "SHA256SUMS"), []byte(checksums.String()), 0644); err != nil {
		return err
	}
	fmt.Println("Release manifest and SHA256SUMS written to", directory)
	return nil
}
