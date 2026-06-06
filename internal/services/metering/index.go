package metering

import (
	"context"

	"metering-service/internal/meter"
)

type Service interface {
	TrackRequest(ctx context.Context, endpoint string) (*TrackResponse, error)
	Metrics(ctx context.Context) (*MetricsResponse, error)
	RecordUpload(ctx context.Context, filename string, size int64) (*UploadResponse, error)
	Storage(ctx context.Context) (*StorageResponse, error)
}

type service struct {
	store *meter.Storage
}

func NewService(store *meter.Storage) Service {
	return &service{store: store}
}
