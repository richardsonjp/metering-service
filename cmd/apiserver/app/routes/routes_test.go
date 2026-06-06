package routes_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/cmd/apiserver/app/routes"
	"metering-service/cmd/apiserver/app/store"
	"metering-service/config"
)

func newTestApp(t *testing.T, cfg config.Config) (*fiber.App, func()) {
	t.Helper()
	appStore := store.Init(cfg)
	return routes.NewHTTPServer(appStore), appStore.Shutdown
}

func defaultTestConfig() config.Config {
	return config.Config{
		System: config.System{Addr: ":0"},
		Metering: config.Metering{
			RequestLimit:   1000,
			StorageLimit:   1 << 30,
			MaxUploadBytes: 1 << 30,
		},
	}
}

func doJSON(t *testing.T, app *fiber.App, req *http.Request, out interface{}) *http.Response {
	t.Helper()
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	if out != nil {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, out), "body: %s", string(body))
	}
	return resp
}

func multipartFile(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func pngBytes(n int) []byte {
	sig := []byte("\x89PNG\r\n\x1a\n")
	if n <= len(sig) {
		return sig
	}
	return append(sig, make([]byte, n-len(sig))...)
}

func mp4Bytes() []byte {
	return []byte{0x00, 0x00, 0x00, 0x10, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0}
}

func TestHealth(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	resp := doJSON(t, app, httptest.NewRequest(http.MethodGet, "/health", nil), nil)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestTrackEndpointAndMetrics(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	for i := 0; i < 2; i++ {
		var out struct {
			Data struct {
				Endpoint string `json:"endpoint"`
				Count    int64  `json:"count"`
			} `json:"data"`
		}
		resp := doJSON(t, app, httptest.NewRequest(http.MethodPost, "/api/endpoint1", nil), &out)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, "/api/endpoint1", out.Data.Endpoint)
		assert.Equal(t, int64(i+1), out.Data.Count)
	}

	var metrics struct {
		Data struct {
			Endpoints map[string]int64 `json:"endpoints"`
			Total     int64            `json:"total_requests"`
			Remaining int64            `json:"remaining"`
		} `json:"data"`
	}
	resp := doJSON(t, app, httptest.NewRequest(http.MethodGet, "/api/metrics", nil), &metrics)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(2), metrics.Data.Endpoints["/api/endpoint1"])
	assert.Equal(t, int64(2), metrics.Data.Total)
	assert.Equal(t, int64(998), metrics.Data.Remaining)
}

func TestRequestLimitReturns429(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Metering.RequestLimit = 2
	app, cleanup := newTestApp(t, cfg)
	defer cleanup()

	codes := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/endpoint1", nil), -1)
		require.NoError(t, err)
		codes = append(codes, resp.StatusCode)
	}
	assert.Equal(t, []int{fiber.StatusOK, fiber.StatusOK, fiber.StatusTooManyRequests}, codes)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/metrics", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestUploadAndStorage(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	body, contentType := multipartFile(t, "file", "photo.jpg", pngBytes(1024))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(fiber.HeaderContentType, contentType)

	var up struct {
		Data struct {
			Filename       string `json:"filename"`
			Size           int64  `json:"size"`
			TotalUsedBytes int64  `json:"total_used_bytes"`
		} `json:"data"`
	}
	resp := doJSON(t, app, req, &up)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	assert.Equal(t, "photo.jpg", up.Data.Filename)
	assert.Equal(t, int64(1024), up.Data.Size)
	assert.Equal(t, int64(1024), up.Data.TotalUsedBytes)

	var st struct {
		Data struct {
			UsedBytes      int64 `json:"used_bytes"`
			RemainingBytes int64 `json:"remaining_bytes"`
		} `json:"data"`
	}
	resp = doJSON(t, app, httptest.NewRequest(http.MethodGet, "/storage", nil), &st)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(1024), st.Data.UsedBytes)
	assert.Equal(t, (1<<30)-int64(1024), st.Data.RemainingBytes)
}

func TestUploadMissingFileReturns400(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(""))
	req.Header.Set(fiber.HeaderContentType, "multipart/form-data; boundary=none")

	var out struct {
		Code string `json:"code"`
	}
	resp := doJSON(t, app, req, &out)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "FILE_REQUIRED", out.Code)
}

func TestUploadTooLargeReturns413(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Metering.MaxUploadBytes = 50
	app, cleanup := newTestApp(t, cfg)
	defer cleanup()

	body, contentType := multipartFile(t, "file", "big.bin", pngBytes(100))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(fiber.HeaderContentType, contentType)

	var out struct {
		Code string `json:"code"`
	}
	resp := doJSON(t, app, req, &out)
	assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)
	assert.Equal(t, "FILE_TOO_LARGE", out.Code)
}

func TestUploadOverStorageCapReturns507(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Metering.StorageLimit = 100
	cfg.Metering.MaxUploadBytes = 1000
	app, cleanup := newTestApp(t, cfg)
	defer cleanup()

	body, contentType := multipartFile(t, "file", "big.bin", pngBytes(150))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(fiber.HeaderContentType, contentType)

	var out struct {
		Code string `json:"code"`
	}
	resp := doJSON(t, app, req, &out)
	assert.Equal(t, fiber.StatusInsufficientStorage, resp.StatusCode)
	assert.Equal(t, "STORAGE_LIMIT_EXCEEDED", out.Code)
}

func TestUploadRejectsNonMediaReturns415(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	body, contentType := multipartFile(t, "file", "notes.txt",
		[]byte("this is plain text, not an image or a video at all"))
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(fiber.HeaderContentType, contentType)

	var out struct {
		Code string `json:"code"`
	}
	resp := doJSON(t, app, req, &out)
	assert.Equal(t, fiber.StatusUnsupportedMediaType, resp.StatusCode)
	assert.Equal(t, "UNSUPPORTED_FILE_TYPE", out.Code)
}

func TestUploadAcceptsVideo(t *testing.T) {
	app, cleanup := newTestApp(t, defaultTestConfig())
	defer cleanup()

	body, contentType := multipartFile(t, "file", "clip.mp4", mp4Bytes())
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(fiber.HeaderContentType, contentType)

	var up struct {
		Data struct {
			Filename string `json:"filename"`
		} `json:"data"`
	}
	resp := doJSON(t, app, req, &up)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	assert.Equal(t, "clip.mp4", up.Data.Filename)
}
