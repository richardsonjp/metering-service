package meter

import (
	"math"
	"sync"
	"sync/atomic"

	"metering-service/pkg/utils/logs"
)

type Storage struct {
	requestLimit int64
	storageLimit int64

	totalRequests atomic.Int64

	mu        sync.RWMutex
	endpoints map[string]*atomic.Int64

	storageCmds chan storageCmd
	storageDone chan struct{}
	storageStop sync.Once
}

type storageCmd struct {
	size  int64
	query bool
	reset bool
	reply chan storageReply
}

type storageReply struct {
	used int64
	ok   bool
}

type RequestSnapshot struct {
	Endpoints map[string]int64
	Total     int64
	Limit     int64
	Remaining int64
}

type StorageSnapshot struct {
	Used      int64
	Limit     int64
	Remaining int64
}

func NewStorage(requestLimit, storageLimit int64) *Storage {
	s := &Storage{
		requestLimit: requestLimit,
		storageLimit: storageLimit,
		endpoints:    make(map[string]*atomic.Int64),
		storageCmds:  make(chan storageCmd),
		storageDone:  make(chan struct{}),
	}
	go s.runStorage()
	return s
}

func (s *Storage) Close() {
	s.storageStop.Do(func() { close(s.storageDone) })
}

func (s *Storage) runStorage() {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("storage actor recovered from panic", logs.Fields{"panic": r})
		}
	}()

	var used int64
	for {
		select {
		case cmd := <-s.storageCmds:
			switch {
			case cmd.reset:
				used = 0
				cmd.reply <- storageReply{used: 0, ok: true}
			case cmd.query:
				cmd.reply <- storageReply{used: used, ok: true}
			case cmd.size < 0 || cmd.size > math.MaxInt64-used:
				cmd.reply <- storageReply{used: used, ok: false}
			case s.storageLimit > 0 && used+cmd.size > s.storageLimit:
				cmd.reply <- storageReply{used: used, ok: false}
			default:
				used += cmd.size
				cmd.reply <- storageReply{used: used, ok: true}
			}
		case <-s.storageDone:
			return
		}
	}
}

func (s *Storage) askStorage(cmd storageCmd) storageReply {
	cmd.reply = make(chan storageReply, 1)
	s.storageCmds <- cmd
	return <-cmd.reply
}

func (s *Storage) RecordRequest(endpoint string) (count, total int64, ok bool) {
	total = s.totalRequests.Add(1)
	if s.requestLimit > 0 && total > s.requestLimit {
		s.totalRequests.Add(-1)
		return 0, s.requestLimit, false
	}
	count = s.counterFor(endpoint).Add(1)
	return count, total, true
}

func (s *Storage) ReserveStorage(size int64) (totalUsed int64, ok bool) {
	r := s.askStorage(storageCmd{size: size})
	return r.used, r.ok
}

func (s *Storage) RequestSnapshot() RequestSnapshot {
	s.mu.RLock()
	eps := make(map[string]int64, len(s.endpoints))
	for k, v := range s.endpoints {
		eps[k] = v.Load()
	}
	s.mu.RUnlock()

	total := s.totalRequests.Load()
	return RequestSnapshot{
		Endpoints: eps,
		Total:     total,
		Limit:     s.requestLimit,
		Remaining: Remaining(s.requestLimit, total),
	}
}

func (s *Storage) StorageSnapshot() StorageSnapshot {
	used := s.askStorage(storageCmd{query: true}).used
	return StorageSnapshot{
		Used:      used,
		Limit:     s.storageLimit,
		Remaining: Remaining(s.storageLimit, used),
	}
}

func (s *Storage) RequestLimit() int64 { return s.requestLimit }

func (s *Storage) StorageLimit() int64 { return s.storageLimit }

func (s *Storage) Reset() {
	s.mu.Lock()
	s.endpoints = make(map[string]*atomic.Int64)
	s.mu.Unlock()
	s.totalRequests.Store(0)
	s.askStorage(storageCmd{reset: true})
}

func (s *Storage) counterFor(endpoint string) *atomic.Int64 {
	s.mu.RLock()
	c, ok := s.endpoints[endpoint]
	s.mu.RUnlock()
	if ok {
		return c
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok = s.endpoints[endpoint]; ok {
		return c
	}
	c = new(atomic.Int64)
	s.endpoints[endpoint] = c
	return c
}

func Remaining(limit, used int64) int64 {
	if limit <= 0 {
		return -1
	}
	if r := limit - used; r > 0 {
		return r
	}
	return 0
}
