package metering_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/internal/meter"
	"metering-service/internal/services/metering"
	"metering-service/pkg/utils/errors"
)

func newService(requestLimit, storageLimit int64) (metering.Service, func()) {
	store := meter.NewStorage(requestLimit, storageLimit)
	return metering.NewService(store), store.Close
}

func TestService_TrackRequest_HappyPath(t *testing.T) {
	svc, cleanup := newService(5, 0)
	defer cleanup()

	res, err := svc.TrackRequest(context.Background(), "/api/endpoint1")
	require.NoError(t, err)
	assert.Equal(t, "/api/endpoint1", res.Endpoint)
	assert.Equal(t, int64(1), res.Count)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, int64(4), res.Remaining)
}

func TestService_TrackRequest_LimitExceeded(t *testing.T) {
	svc, cleanup := newService(1, 0)
	defer cleanup()

	_, err := svc.TrackRequest(context.Background(), "/api/endpoint1")
	require.NoError(t, err)

	_, err = svc.TrackRequest(context.Background(), "/api/endpoint1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, "REQUEST_LIMIT_EXCEEDED"))
}

func TestService_Metrics(t *testing.T) {
	svc, cleanup := newService(10, 0)
	defer cleanup()

	_, _ = svc.TrackRequest(context.Background(), "/api/endpoint1")
	_, _ = svc.TrackRequest(context.Background(), "/api/endpoint1")
	_, _ = svc.TrackRequest(context.Background(), "/upload")

	res, err := svc.Metrics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Endpoints["/api/endpoint1"])
	assert.Equal(t, int64(1), res.Endpoints["/upload"])
	assert.Equal(t, int64(3), res.Total)
	assert.Equal(t, int64(10), res.Limit)
	assert.Equal(t, int64(7), res.Remaining)
}

func TestService_RecordUpload_HappyPath(t *testing.T) {
	svc, cleanup := newService(0, 1000)
	defer cleanup()

	res, err := svc.RecordUpload(context.Background(), "photo.jpg", 250)
	require.NoError(t, err)
	assert.Equal(t, "photo.jpg", res.Filename)
	assert.Equal(t, int64(250), res.Size)
	assert.Equal(t, int64(250), res.TotalUsedBytes)
	assert.Equal(t, int64(750), res.RemainingBytes)
}

func TestService_RecordUpload_EmptyFile(t *testing.T) {
	svc, cleanup := newService(0, 1000)
	defer cleanup()

	_, err := svc.RecordUpload(context.Background(), "empty.bin", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, "FILE_REQUIRED"))
}

func TestService_RecordUpload_StorageLimitExceeded(t *testing.T) {
	svc, cleanup := newService(0, 100)
	defer cleanup()

	_, err := svc.RecordUpload(context.Background(), "big.bin", 150)
	require.Error(t, err)
	assert.True(t, errors.Is(err, "STORAGE_LIMIT_EXCEEDED"))
}

func TestService_Storage(t *testing.T) {
	svc, cleanup := newService(0, 1000)
	defer cleanup()

	_, _ = svc.RecordUpload(context.Background(), "a.bin", 300)

	res, err := svc.Storage(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(300), res.UsedBytes)
	assert.Equal(t, int64(1000), res.LimitBytes)
	assert.Equal(t, int64(700), res.RemainingBytes)
	assert.NotEmpty(t, res.UsedHuman)
}

func TestService_TrackRequest_RemainingCoherentUnderConcurrency(t *testing.T) {
	const (
		limit      = 500
		goroutines = 2000
	)
	svc, cleanup := newService(limit, 0)
	defer cleanup()

	var wg sync.WaitGroup
	bad := make(chan string, goroutines)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			res, err := svc.TrackRequest(context.Background(), "/api/endpoint1")
			if err != nil {
				return
			}
			if res.Total+res.Remaining != int64(limit) {
				bad <- fmt.Sprintf("total=%d remaining=%d", res.Total, res.Remaining)
			}
		}()
	}
	wg.Wait()
	close(bad)

	for msg := range bad {
		t.Errorf("incoherent response %s (want total+remaining=%d)", msg, limit)
	}
}
