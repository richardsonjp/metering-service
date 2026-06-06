package metering

type TrackResponse struct {
	Endpoint  string `json:"endpoint"`
	Count     int64  `json:"count"`
	Total     int64  `json:"total"`
	Remaining int64  `json:"remaining"`
}

type MetricsResponse struct {
	Endpoints map[string]int64 `json:"endpoints"`
	Total     int64            `json:"total_requests"`
	Limit     int64            `json:"limit"`
	Remaining int64            `json:"remaining"`
}

type UploadResponse struct {
	Filename       string `json:"filename"`
	Size           int64  `json:"size"`
	SizeHuman      string `json:"size_human"`
	TotalUsedBytes int64  `json:"total_used_bytes"`
	TotalUsedHuman string `json:"total_used_human"`
	RemainingBytes int64  `json:"remaining_bytes"`
}

type StorageResponse struct {
	UsedBytes      int64  `json:"used_bytes"`
	UsedHuman      string `json:"used_human"`
	LimitBytes     int64  `json:"limit_bytes"`
	LimitHuman     string `json:"limit_human"`
	RemainingBytes int64  `json:"remaining_bytes"`
	RemainingHuman string `json:"remaining_human"`
}
