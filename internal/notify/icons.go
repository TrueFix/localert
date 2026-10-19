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
	img := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	
	// Background colors
	var bg color.NRGBA
	switch kind {
	case "success":
		bg = color.NRGBA{R: 34, G: 197, B: 94, A: 255} // Green
	case "warning":
		bg = color.NRGBA{R: 245, G: 158, B: 11, A: 255} // Amber
	case "error":
		bg = color.NRGBA{R: 239, G: 68, B: 68, A: 255} // Red
	case "info":
		bg = color.NRGBA{R: 59, G: 130, B: 246, A: 255} // Blue
	default: // bell
		bg = color.NRGBA{R: 139, G: 92, B: 246, A: 255} // Purple
	}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	
	// Draw rounded rectangle background
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			// Rounded corners with radius 24
			r := 24
			inCorner := false
			corners := [][2]int{{r, r}, {127 - r, r}, {r, 127 - r}, {127 - r, 127 - r}}
			for _, c := range corners {
				dx, dy := x-c[0], y-c[1]
				if (x < r || x > 127-r) && (y < r || y > 127-r) {
					if dx*dx+dy*dy > r*r {
						inCorner = true
					}
				}
			}
			if !inCorner {
				img.SetNRGBA(x, y, bg)
			}
		}
	}
	
	// Draw icon symbol
	switch kind {
	case "success": // Checkmark
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				// Left part of check (going down-right)
				if x >= 30 && x <= 55 && y >= 50 && y <= 85 {
					if abs(y-x-20) < 8 {
						img.SetNRGBA(x, y, white)
					}
				}
				// Right part of check (going up-right)
				if x >= 50 && x <= 98 && y >= 35 && y <= 75 {
					if abs(y+x-125) < 8 {
						img.SetNRGBA(x, y, white)
					}
				}
			}
		}
	case "warning": // Exclamation mark
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				cx := 64
				// Vertical line
				if x >= cx-6 && x <= cx+6 && y >= 30 && y <= 75 {
					img.SetNRGBA(x, y, white)
				}
				// Dot
				dx, dy := x-cx, y-90
				if dx*dx+dy*dy <= 8*8 {
					img.SetNRGBA(x, y, white)
				}
			}
		}
	case "error": // X mark
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				if x >= 35 && x <= 93 && y >= 35 && y <= 93 {
					// Diagonal line 1
					if abs(x-y) < 8 {
						img.SetNRGBA(x, y, white)
					}
					// Diagonal line 2
					if abs((x-64)+(y-64)) < 8 {
						img.SetNRGBA(x, y, white)
					}
				}
			}
		}
	case "info": // Letter i
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				cx := 64
				// Dot on top
				dx, dy := x-cx, y-35
				if dx*dx+dy*dy <= 8*8 {
					img.SetNRGBA(x, y, white)
				}
				// Vertical line
				if x >= cx-6 && x <= cx+6 && y >= 50 && y <= 95 {
					img.SetNRGBA(x, y, white)
				}
			}
		}
	default: // bell
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				cx := 64
				// Bell body (dome)
				if y >= 30 && y <= 80 {
					// Parabolic bell shape
					width := 30 + (y-30)*25/50
					if abs(x-cx) <= width && y <= 80 {
						img.SetNRGBA(x, y, white)
					}
				}
				// Bell bottom rim
				if y >= 80 && y <= 88 && x >= 24 && x <= 104 {
					img.SetNRGBA(x, y, white)
				}
				// Clapper
				dx, dy := x-cx, y-98
				if dx*dx+dy*dy <= 8*8 {
					img.SetNRGBA(x, y, white)
				}
			}
		}
	}
	return img
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
