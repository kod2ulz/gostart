package api

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// FileRequest represents an uploaded file with metadata and validation capabilities.
// It implements contracts.RequestParam interface so it can be used as a typed parameter
// in TypedHandler functions.
//
// Example usage:
//
//	type UploadAvatarRequest struct {
//	    api.RequestModal[UploadAvatarRequest]
//	    api.FileRequest
//	    UserID string `param:"userId"`
//	}
//
//	func (h *Handler) UploadAvatar(ctx contracts.RequestContext, req UploadAvatarRequest) (string, ierrors.Error) {
//	    // Validate file type and size
//	    if err := req.File.ValidateFileType("image/jpeg", "image/png").ValidateMaxSize(5 * 1024 * 1024); err != nil {
//	        return "", err
//	    }
//
//	    // Save to disk
//	    destPath := "/avatars/" + req.UserID + ".jpg"
//	    if err := req.SaveToFile(destPath); err != nil {
//	        return "", err
//	    }
//
//	    return destPath, nil
//	}
type FileRequest struct {
	// File is the multipart file header from the request
	File *multipart.FileHeader `file:"file"`

	// Filename is the original filename of the uploaded file
	Filename string

	// ContentType is the MIME type of the file
	ContentType string

	// Size is the file size in bytes
	Size int64

	// maxFileSize is the maximum allowed file size (0 = unlimited)
	maxFileSize int64

	// allowedContentTypes is a list of allowed MIME types (empty = any type)
	allowedContentTypes []string

	// reader is the file reader for streaming
	reader io.Reader
}

// FileHeader is the interface for multipart file headers
type FileHeader interface {
	Open() (multipart.File, error)
}

// Validate validates the file request
func (f *FileRequest) Validate(ctx contracts.RequestContext) error {
	if f.File == nil {
		return errors.ValidationFailed[FileRequest](fmt.Errorf("file is required"))
	}

	// Validate file size
	if f.maxFileSize > 0 && f.File.Size > f.maxFileSize {
		err := fmt.Errorf("file size %d exceeds maximum allowed size of %d bytes", f.File.Size, f.maxFileSize)
		return errors.ValidationFailed[FileRequest](err).
			WithMessage("file size exceeds maximum allowed size")
	}

	// Validate content type
	if len(f.allowedContentTypes) > 0 {
		allowed := false
		for _, ct := range f.allowedContentTypes {
			if f.ContentType == ct {
				allowed = true
				break
			}
		}
		if !allowed {
			err := fmt.Errorf("file type %s is not allowed. Allowed types: %s", f.ContentType, strings.Join(f.allowedContentTypes, ", "))
			return errors.ValidationFailed[FileRequest](err).
				WithMessage("file type not allowed")
		}
	}

	return nil
}

// ValidateFileType sets the allowed content types and returns the FileRequest for chaining
func (f *FileRequest) ValidateFileType(contentTypes ...string) *FileRequest {
	f.allowedContentTypes = contentTypes
	return f
}

// ValidateMaxSize sets the maximum file size in bytes and returns the FileRequest for chaining
func (f *FileRequest) ValidateMaxSize(maxBytes int64) *FileRequest {
	f.maxFileSize = maxBytes
	return f
}

// ValidateMaxSizeMB sets the maximum file size in megabytes and returns the FileRequest for chaining
func (f *FileRequest) ValidateMaxSizeMB(maxMB int64) *FileRequest {
	f.maxFileSize = maxMB * 1024 * 1024
	return f
}

// Open opens the file for reading
func (f *FileRequest) Open() (multipart.File, error) {
	if f.File == nil {
		return nil, fmt.Errorf("file is nil")
	}
	return f.File.Open()
}

// Reader returns a reader for the file content
func (f *FileRequest) Reader() (io.Reader, error) {
	if f.reader != nil {
		return f.reader, nil
	}

	file, err := f.Open()
	if err != nil {
		return nil, err
	}

	return file, nil
}

// SaveToFile saves the uploaded file to the specified path
func (f *FileRequest) SaveToFile(destPath string) ierrors.Error {
	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create directory")
	}

	// Open source file
	src, err := f.Open()
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to open uploaded file")
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create destination file")
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to save file")
	}

	return nil
}

// SaveToFileWithPermissions saves the uploaded file with specific permissions
func (f *FileRequest) SaveToFileWithPermissions(destPath string, perm os.FileMode) ierrors.Error {
	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create directory")
	}

	// Open source file
	src, err := f.Open()
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to open uploaded file")
	}
	defer src.Close()

	// Create destination file with permissions
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create destination file")
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to save file")
	}

	return nil
}

// PipeToWriter pipes the file content to an io.Writer
// This is useful for streaming directly to MinIO, Kafka, or other services
func (f *FileRequest) PipeToWriter(dst io.Writer) ierrors.Error {
	src, err := f.Open()
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to open uploaded file")
	}
	defer src.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to pipe file content")
	}

	return nil
}

// PipeToMinIO uploads the file to MinIO (example - would need MinIO client)
// func (f *FileRequest) PipeToMinIO(minioClient *minio.Client, bucketName, objectName string) ierrors.Error {
//     src, err := f.Open()
//     if err != nil {
//         return errors.InternalError(err).WithMessage("failed to open uploaded file")
//     }
//     defer src.Close()
//
//     _, err = minioClient.PutObject(context.Background(), bucketName, objectName, src, f.Size, minio.PutObjectOptions{
//         ContentType: f.ContentType,
//     })
//
//     if err != nil {
//         return errors.InternalError(err).WithMessage("failed to upload to MinIO")
//     }
//
//     return nil
// }

// PipeToKafka produces the file to Kafka (example - would need Kafka client)
// This would typically be for metadata only, not the actual file content
// For large files, you'd upload to storage and send the URL via Kafka
// func (f *FileRequest) PipeToKafka(producer kafka.Producer, topic string) ierrors.Error {
//     message := &kafka.ProducerMessage{
//         Topic: topic,
//         Value: kafka.ByteEncoder(fmt.Sprintf(`{"filename":"%s","size":%d,"type":"%s"}`,
//             f.Filename, f.Size, f.ContentType)),
//     }
//
//     if err := producer.Produce(message, nil); err != nil {
//         return errors.InternalError(err).WithMessage("failed to send file metadata to Kafka")
//     }
//
//     return nil
// }

// PipeToAPI sends the file to another API endpoint
func (f *FileRequest) PipeToAPI(client *http.Client, apiURL, authHeader string) ierrors.Error {
	src, err := f.Open()
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to open uploaded file")
	}
	defer src.Close()

	// Create a multipart form
	body := &strings.Builder{}
	writer := multipart.NewWriter(body)

	// Create form file field
	part, err := writer.CreateFormFile("file", f.Filename)
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create form file")
	}

	// Copy file content to form field
	if _, err := io.Copy(part, src); err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to copy file content")
	}

	// Close writer to finalize form
	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body.String()))
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to create request")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return errors.GeneralFailure[FileRequest](err).WithMessage("failed to send request to API")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.GeneralFailure[FileRequest](fmt.Errorf("API returned status %d", resp.StatusCode)).
			WithMessage("failed to upload file to API")
	}

	return nil
}

// ContextKey returns the key for storing this file request in context
func (f *FileRequest) ContextKey() string {
	return "FileRequest"
}

// RequestLoad loads the file from the HTTP request context
func (f *FileRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
	// Try to get file from context (if already loaded by middleware)
	if ctxGetter, ok := ctx.(interface{ Value(interface{}) interface{} }); ok {
		if val := ctxGetter.Value("uploaded_file"); val != nil {
			if fileHeader, ok := val.(*multipart.FileHeader); ok {
				f.File = fileHeader
				f.Filename = fileHeader.Filename
				f.Size = fileHeader.Size
				f.ContentType = http.DetectContentType([]byte(fileHeader.Filename))
				return f, nil
			}
		}
	}

	return nil, errors.ValidationFailed[FileRequest](fmt.Errorf("no file uploaded"))
}

// ContextLoad loads the file from a standard Go context
func (f *FileRequest) ContextLoad(ctx context.Context) (contracts.RequestParam, error) {
	val := ctx.Value(f.ContextKey())
	if val == nil {
		return nil, fmt.Errorf("file not found in context")
	}

	fileReq, ok := val.(*FileRequest)
	if !ok {
		return nil, fmt.Errorf("invalid file request type in context")
	}

	return fileReq, nil
}

// GetExtension returns the file extension from the filename
func (f *FileRequest) GetExtension() string {
	ext := filepath.Ext(f.Filename)
	if ext != "" {
		return strings.TrimPrefix(ext, ".")
	}
	return ""
}

// GetContentType returns the MIME type based on file extension
// This is a basic implementation - for production use a proper MIME type library
func (f *FileRequest) GetContentType() string {
	ext := strings.ToLower(f.GetExtension())

	mimeTypes := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"pdf":  "application/pdf",
		"doc":  "application/msword",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xls":  "application/vnd.ms-excel",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"txt":  "text/plain",
		"csv":  "text/csv",
		"json": "application/json",
		"xml":  "application/xml",
		"zip":  "application/zip",
		"mp3":  "audio/mpeg",
		"mp4":  "video/mp4",
	}

	if ct, ok := mimeTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}
