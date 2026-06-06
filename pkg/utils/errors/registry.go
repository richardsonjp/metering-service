package errors

import "net/http"

var Registry = map[string]AppError{
	"BAD_REQUEST":           {Code: "BAD_REQUEST", Status: http.StatusBadRequest, Message: "Bad request"},
	"VALIDATION_FAILED":     {Code: "VALIDATION_FAILED", Status: http.StatusUnprocessableEntity, Message: "Validation failed"},
	"INTERNAL_SERVER_ERROR": {Code: "INTERNAL_SERVER_ERROR", Status: http.StatusInternalServerError, Message: "Internal server error"},
	"DATA_NOT_FOUND":        {Code: "DATA_NOT_FOUND", Status: http.StatusNotFound, Message: "Data not found"},

	"REQUEST_LIMIT_EXCEEDED": {Code: "REQUEST_LIMIT_EXCEEDED", Status: http.StatusTooManyRequests, Message: "API request limit reached; access is stopped"},
	"STORAGE_LIMIT_EXCEEDED": {Code: "STORAGE_LIMIT_EXCEEDED", Status: http.StatusInsufficientStorage, Message: "Storage limit reached; upload rejected"},
	"FILE_REQUIRED":          {Code: "FILE_REQUIRED", Status: http.StatusBadRequest, Message: "A non-empty file upload is required"},
	"FILE_TOO_LARGE":         {Code: "FILE_TOO_LARGE", Status: http.StatusRequestEntityTooLarge, Message: "Uploaded file exceeds the per-file size limit"},
	"UNSUPPORTED_FILE_TYPE":  {Code: "UNSUPPORTED_FILE_TYPE", Status: http.StatusUnsupportedMediaType, Message: "Only image and video uploads are allowed"},
}
