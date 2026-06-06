package metering

import (
	"context"
	"fmt"

	"metering-service/internal/meter"
	"metering-service/pkg/utils/bytesize"
	"metering-service/pkg/utils/errors"
)

func (s *service) TrackRequest(_ context.Context, endpoint string) (*TrackResponse, error) {
	count, total, ok := s.store.RecordRequest(endpoint)
	if !ok {
		return nil, errors.From("REQUEST_LIMIT_EXCEEDED").
			WithDetail(fmt.Sprintf("global request limit reached for %q", endpoint))
	}

	return &TrackResponse{
		Endpoint: endpoint,
		Count:    count,
		Total:    total,

		Remaining: meter.Remaining(s.store.RequestLimit(), total),
	}, nil
}

func (s *service) Metrics(_ context.Context) (*MetricsResponse, error) {
	snap := s.store.RequestSnapshot()
	return &MetricsResponse{
		Endpoints: snap.Endpoints,
		Total:     snap.Total,
		Limit:     snap.Limit,
		Remaining: snap.Remaining,
	}, nil
}

func (s *service) RecordUpload(_ context.Context, filename string, size int64) (*UploadResponse, error) {
	if size <= 0 {
		return nil, errors.From("FILE_REQUIRED").WithDetail("uploaded file is empty")
	}

	total, ok := s.store.ReserveStorage(size)
	if !ok {
		return nil, errors.From("STORAGE_LIMIT_EXCEEDED").WithDetail(
			fmt.Sprintf("uploading %s would exceed the %s storage limit (%s already used)",
				bytesize.Human(size), bytesize.Human(s.store.StorageLimit()), bytesize.Human(total)),
		)
	}

	return &UploadResponse{
		Filename:       filename,
		Size:           size,
		SizeHuman:      bytesize.Human(size),
		TotalUsedBytes: total,
		TotalUsedHuman: bytesize.Human(total),

		RemainingBytes: meter.Remaining(s.store.StorageLimit(), total),
	}, nil
}

func (s *service) Storage(_ context.Context) (*StorageResponse, error) {
	snap := s.store.StorageSnapshot()
	return &StorageResponse{
		UsedBytes:      snap.Used,
		UsedHuman:      bytesize.Human(snap.Used),
		LimitBytes:     snap.Limit,
		LimitHuman:     bytesize.Human(snap.Limit),
		RemainingBytes: snap.Remaining,
		RemainingHuman: bytesize.Human(snap.Remaining),
	}, nil
}
