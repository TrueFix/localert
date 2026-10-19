package notify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxIconBytes = 1 << 20

// Icons normalizes built-in icons, HTTPS images, and data URLs into private PNGs.
// Cached paths survive delivery, as desktop systems may read an icon asynchronously.
type Icons struct {
	dir    string
	client *http.Client
}

func NewIcons(dataDir string) (*Icons, error) {
	dir := filepath.Join(dataDir, "icons")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Icons{dir: dir, client: &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many icon redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("icon redirects must use HTTPS")
			}
			return nil
		},
	}}, nil
}

func (icons *Icons) Resolve(ctx context.Context, icon string) (string, error) {
	if icon == "" {
		icon = "bell"
	}
	var img image.Image
	switch icon {
	case "bell", "info", "success", "warning", "error":
		img = builtinIcon(icon)
	default:
		var data []byte
		var err error
		if strings.HasPrefix(icon, "data:image/png;base64,") || strings.HasPrefix(icon, "data:image/jpeg;base64,") {
			_, payload, _ := strings.Cut(icon, ",")
			if len(payload) > base64.StdEncoding.EncodedLen(maxIconBytes) {
				return "", errors.New("icon exceeds 1 MiB")
			}
			data, err = base64.StdEncoding.DecodeString(payload)
		} else {
			u, parseErr := url.Parse(icon)
			if parseErr != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
				return "", errors.New("icon must be bell, info, success, warning, error, an HTTPS image URL, or a base64 PNG/JPEG data URL")
			}
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if reqErr != nil {
				return "", reqErr
			}
			resp, fetchErr := icons.client.Do(req)
			if fetchErr != nil {
				return "", fmt.Errorf("fetch icon: %w", fetchErr)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("icon server returned HTTP %d", resp.StatusCode)
			}
			data, err = io.ReadAll(io.LimitReader(resp.Body, maxIconBytes+1))
		}
		if err != nil {
			return "", fmt.Errorf("read icon: %w", err)
		}
		if len(data) > maxIconBytes {
			return "", errors.New("icon exceeds 1 MiB")
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 1024 || config.Height > 1024 {
			return "", errors.New("icon must be a valid PNG or JPEG up to 1024 × 1024 pixels")
		}
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("decode icon: %w", err)
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		return "", err
	}
	digest := sha256.Sum256(output.Bytes())
	path := filepath.Join(icons.dir, hex.EncodeToString(digest[:])+".png")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	f, err := os.CreateTemp(icons.dir, ".icon-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err := f.Write(output.Bytes()); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		// Another request may have populated the same content-addressed cache entry.
		if _, statErr := os.Stat(path); statErr != nil {
			return "", err
		}
	}
	return path, nil
}

// The small default icon is generated in-process, so it is embedded in the executable too.
func builtinIcon(kind string) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	fill := color.NRGBA{R: 38, G: 38, B: 38, A: 255}
	switch kind {
	case "success":
		fill = color.NRGBA{R: 35, G: 130, B: 80, A: 255}
	case "warning":
		fill = color.NRGBA{R: 185, G: 120, B: 15, A: 255}
	case "error":
		fill = color.NRGBA{R: 200, G: 60, B: 60, A: 255}
	case "info":
		fill = color.NRGBA{R: 45, G: 110, B: 200, A: 255}
	}
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			dx, dy := x-32, y-32
			if dx*dx+dy*dy < 30*30 {
				img.SetNRGBA(x, y, fill)
			}
			white := (y >= 27 && y <= 43 && x >= 20 && x <= 44) || (y >= 19 && y < 27 && (x-32)*(x-32)+(y-28)*(y-28) <= 12*12) || (y >= 44 && y <= 47 && x >= 17 && x <= 47) || (y >= 50 && y <= 53 && x >= 29 && x <= 35)
			if white {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	return img
}
