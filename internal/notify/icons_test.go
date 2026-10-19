package notify

import (
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestIconsNormalizeBuiltinsDataAndHTTPSImages(t *testing.T) {
	icons, err := NewIcons(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path, err := icons.Resolve(context.Background(), "success")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(strings.NewReader(string(data))); err != nil {
		t.Fatalf("built-in icon is not a PNG: %v", err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	dataPath, err := icons.Resolve(context.Background(), dataURL)
	if err != nil || dataPath != path {
		t.Fatalf("data URL did not normalize to same icon: %s, %v", dataPath, err)
	}
	imageServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer imageServer.Close()
	icons.client = imageServer.Client()
	urlPath, err := icons.Resolve(context.Background(), imageServer.URL+"/icon.png")
	if err != nil || urlPath != path {
		t.Fatalf("HTTPS icon did not normalize: %s, %v", urlPath, err)
	}
}

func TestIconsRejectInvalidOrOversizedImages(t *testing.T) {
	icons, err := NewIcons(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"/etc/passwd", "file:///tmp/icon.png", "http://example.com/icon.png", "data:image/png;base64,bm90LWFuLWltYWdl", "data:image/png;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(maxIconBytes)+4)} {
		if _, err := icons.Resolve(context.Background(), value); err == nil {
			t.Errorf("accepted invalid icon: %.80s", value)
		}
	}
	var output strings.Builder
	if err := png.Encode(&output, image.NewNRGBA(image.Rect(0, 0, 1025, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := icons.Resolve(context.Background(), "data:image/png;base64,"+base64.StdEncoding.EncodeToString([]byte(output.String()))); err == nil {
		t.Fatal("accepted oversized pixel dimensions")
	}
}
