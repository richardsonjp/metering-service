package storage

import (
	"fmt"
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"

	"metering-service/pkg/utils/api"
	"metering-service/pkg/utils/bytesize"
	"metering-service/pkg/utils/errors"
	"metering-service/pkg/utils/media"
)

func (h *Handler) Upload(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return errors.Respond(c, errors.From("FILE_REQUIRED").
			WithDetail("multipart form field 'file' is required"))
	}
	if fileHeader.Size <= 0 {
		return errors.Respond(c, errors.From("FILE_REQUIRED").
			WithDetail("uploaded file is empty"))
	}
	if h.maxUploadBytes > 0 && fileHeader.Size > h.maxUploadBytes {
		return errors.Respond(c, errors.From("FILE_TOO_LARGE").WithDetail(
			fmt.Sprintf("file %s (%s) exceeds the per-file limit of %s",
				fileHeader.Filename, bytesize.Human(fileHeader.Size), bytesize.Human(h.maxUploadBytes))))
	}

	mime, err := sniffMediaType(fileHeader)
	if err != nil {
		return errors.Respond(c, errors.From("BAD_REQUEST").WithDetail("could not read uploaded file"))
	}
	if !media.IsImageOrVideo(mime) {
		return errors.Respond(c, errors.From("UNSUPPORTED_FILE_TYPE").WithDetail(
			fmt.Sprintf("file type %q is not allowed; only images and videos are accepted", mime)))
	}

	res, err := h.meteringService.RecordUpload(c.Context(), fileHeader.Filename, fileHeader.Size)
	if err != nil {
		return errors.Respond(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(api.Base{
		Message: "file recorded",
		Data:    res,
	})
}

func sniffMediaType(fh *multipart.FileHeader) (string, error) {
	f, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	head := make([]byte, media.SniffLen)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	return media.Detect(head[:n]), nil
}
