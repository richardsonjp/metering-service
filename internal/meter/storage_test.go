package meter_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"metering-service/internal/meter"
)

func TestRecordRequest_SingleThreaded(t *testing.T) {
	s := meter.NewStorage(3, 0)
	defer s.Close()

	for i := int64(1); i <= 3; i++ {
		count, total, ok := s.RecordRequest("/api/endpoint1")
		assert.True(t, ok)
		assert.Equal(t, i, count)
		assert.Equal(t, i, total)
	}

	count, _, ok := s.RecordRequest("/api/endpoint1")
	assert.False(t, ok)
	assert.Zero(t, count)

	snap := s.RequestSnapshot()
	assert.Equal(t, int64(3), snap.Total)
	assert.Equal(t, int64(3), snap.Endpoints["/api/endpoint1"])
	assert.Equal(t, int64(0), snap.Remaining)
}

func TestRecordRequest_PerEndpointCounters(t *testing.T) {
	s := meter.NewStorage(0, 0)
	defer s.Close()

	s.RecordRequest("/api/endpoint1")
	s.RecordRequest("/api/endpoint1")
	s.RecordRequest("/upload")

	snap := s.RequestSnapshot()
	assert.Equal(t, int64(2), snap.Endpoints["/api/endpoint1"])
	assert.Equal(t, int64(1), snap.Endpoints["/upload"])
	assert.Equal(t, int64(3), snap.Total)
	assert.Equal(t, int64(-1), snap.Remaining)
}

func TestRecordRequest_ConcurrentCapIsExact(t *testing.T) {
	const (
		limit       = 1000
		goroutines  = 5000
		endpointKey = "/api/endpoint1"
	)
	s := meter.NewStorage(limit, 0)
	defer s.Close()

	var admitted int64
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, _, ok := s.RecordRequest(endpointKey); ok {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	snap := s.RequestSnapshot()
	assert.Equal(t, int64(limit), admitted, "exactly limit requests admitted")
	assert.Equal(t, int64(limit), snap.Total, "global total settles at limit")
	assert.Equal(t, int64(limit), snap.Endpoints[endpointKey], "endpoint count equals admitted")
	assert.Equal(t, int64(0), snap.Remaining)
}

func TestReserveStorage_SingleThreaded(t *testing.T) {
	s := meter.NewStorage(0, 100)
	defer s.Close()

	total, ok := s.ReserveStorage(60)
	assert.True(t, ok)
	assert.Equal(t, int64(60), total)

	total, ok = s.ReserveStorage(60)
	assert.False(t, ok)
	assert.Equal(t, int64(60), total)

	total, ok = s.ReserveStorage(40)
	assert.True(t, ok)
	assert.Equal(t, int64(100), total)

	snap := s.StorageSnapshot()
	assert.Equal(t, int64(100), snap.Used)
	assert.Equal(t, int64(0), snap.Remaining)
}

func TestReserveStorage_NegativeRejected(t *testing.T) {
	s := meter.NewStorage(0, 100)
	defer s.Close()

	_, ok := s.ReserveStorage(-1)
	assert.False(t, ok)
}

func TestReserveStorage_ConcurrentNeverExceeds(t *testing.T) {
	const (
		limit      = 1000
		size       = 30
		goroutines = 500
	)
	s := meter.NewStorage(0, limit)
	defer s.Close()

	var succeeded int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, ok := s.ReserveStorage(size); ok {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	snap := s.StorageSnapshot()
	assert.Equal(t, int64(33), succeeded)
	assert.Equal(t, int64(990), snap.Used)
	assert.LessOrEqual(t, snap.Used, int64(limit), "never exceeds the storage limit")
}

func TestReset(t *testing.T) {
	s := meter.NewStorage(10, 100)
	defer s.Close()

	s.RecordRequest("/api/endpoint1")
	s.ReserveStorage(50)

	s.Reset()

	rs := s.RequestSnapshot()
	ss := s.StorageSnapshot()
	assert.Equal(t, int64(0), rs.Total)
	assert.Empty(t, rs.Endpoints)
	assert.Equal(t, int64(0), ss.Used)
}
