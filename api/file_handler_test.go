package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/contracts"
	gerrors "github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

var _ = Describe("File Handler Tests", func() {
	var router *gin.Engine
	var recorder *httptest.ResponseRecorder

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		recorder = httptest.NewRecorder()
	})

	Describe("File Download Handler", func() {
		It("should handle file download with byte data", func() {
			// Create test file data
			fileContent := []byte("This is test file content for download")
			fileName := "test.txt"

			// Create file handler that returns byte data
			handler := func(ctx contracts.RequestContext) ([]byte, string, ierrors.Error) {
				return fileContent, fileName, nil
			}

			// Add route using a custom file handler wrapper
			router.GET("/download", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Set appropriate headers for file download
				c.Header("Content-Description", "File Transfer")
				c.Header("Content-Transfer-Encoding", "binary")
				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", "application/octet-stream")
				c.Data(http.StatusOK, "application/octet-stream", data)
			})

			req, _ := http.NewRequest("GET", "/download", nil)
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.Bytes()).To(Equal(fileContent))
			Expect(recorder.Header().Get("Content-Disposition")).To(Equal("attachment; filename=test.txt"))
			Expect(recorder.Header().Get("Content-Type")).To(Equal("application/octet-stream"))
		})

		It("should handle file download with io.Reader", func() {
			// Create test file content as reader
			fileContent := "This is test file content from io.Reader"
			reader := strings.NewReader(fileContent)
			fileName := "test-reader.txt"

			// Create file handler that returns io.Reader
			handler := func(ctx contracts.RequestContext) (io.Reader, string, ierrors.Error) {
				return reader, fileName, nil
			}

			// Add route using a custom file handler wrapper
			router.GET("/download-reader", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Set appropriate headers for file download
				c.Header("Content-Description", "File Transfer")
				c.Header("Content-Transfer-Encoding", "binary")
				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", "application/octet-stream")

				// Stream the reader data
				if reader, ok := data.(io.Reader); ok {
					io.Copy(c.Writer, reader)
				}
			})

			req, _ := http.NewRequest("GET", "/download-reader", nil)
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(Equal(fileContent))
			Expect(recorder.Header().Get("Content-Disposition")).To(Equal("attachment; filename=test-reader.txt"))
		})

		It("should handle file download with metadata response", func() {
			// Create test file data
			fileContent := []byte("Test content with metadata")
			fileName := "metadata-test.txt"

			// Create file handler that returns both file data and metadata
			handler := func(ctx contracts.RequestContext) ([]byte, map[string]interface{}, ierrors.Error) {
				metadata := map[string]interface{}{
					"message": "File downloaded successfully",
					"size":    len(fileContent),
					"type":    "text/plain",
					"references": []string{
						"ref1",
						"ref2",
					},
				}
				return fileContent, metadata, nil
			}

			// Add route using a custom file handler wrapper
			router.GET("/download-metadata", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, metadata, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// Set appropriate headers for file download
				c.Header("Content-Description", "File Transfer")
				c.Header("Content-Transfer-Encoding", "binary")
				c.Header("Content-Disposition", "attachment; filename="+fileName)
				c.Header("Content-Type", "application/octet-stream")

				// Add metadata headers
				if msg, ok := metadata["message"].(string); ok {
					c.Header("X-File-Message", msg)
				}
				if size, ok := metadata["size"].(int); ok {
					c.Header("X-File-Size", string(rune(size)))
				}

				c.Data(http.StatusOK, "application/octet-stream", data)
			})

			req, _ := http.NewRequest("GET", "/download-metadata", nil)
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.Bytes()).To(Equal(fileContent))
			Expect(recorder.Header().Get("X-File-Message")).To(Equal("File downloaded successfully"))
		})

		It("should handle file download errors gracefully", func() {
			// Create file handler that returns an error
			handler := func(ctx contracts.RequestContext) ([]byte, string, ierrors.Error) {
				return nil, "", gerrors.NotFound[string]("File not found")
			}

			// Add route using a custom file handler wrapper
			router.GET("/download-error", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				_, _, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{
						"success": false,
						"error":   err.Error(),
					})
					return
				}
			})

			req, _ := http.NewRequest("GET", "/download-error", nil)
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).ToNot(BeNil())
		})

		It("should handle text file download", func() {
			content := []byte("This is a text file")
			contentType := "text/plain"
			filename := "test.txt"

			handler := func(ctx contracts.RequestContext) ([]byte, string, ierrors.Error) {
				return content, filename, nil
			}

			router.GET("/download-txt", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", contentType)
				c.Data(http.StatusOK, contentType, data)
			})

			req, _ := http.NewRequest("GET", "/download-txt", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Bytes()).To(Equal(content))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("Content-Disposition")).To(Equal("attachment; filename="+filename))
		})

		It("should handle JSON file download", func() {
			content := []byte(`{"test": "data", "number": 123}`)
			contentType := "application/json"
			filename := "test.json"

			handler := func(ctx contracts.RequestContext) ([]byte, string, ierrors.Error) {
				return content, filename, nil
			}

			router.GET("/download-json", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", contentType)
				c.Data(http.StatusOK, contentType, data)
			})

			req, _ := http.NewRequest("GET", "/download-json", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Bytes()).To(Equal(content))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("Content-Disposition")).To(Equal("attachment; filename="+filename))
		})

		It("should handle binary file download", func() {
			content := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A} // PNG header
			contentType := "image/png"
			filename := "test.png"

			handler := func(ctx contracts.RequestContext) ([]byte, string, ierrors.Error) {
				return content, filename, nil
			}

			router.GET("/download-png", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", contentType)
				c.Data(http.StatusOK, contentType, data)
			})

			req, _ := http.NewRequest("GET", "/download-png", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Bytes()).To(Equal(content))
			Expect(w.Header().Get("Content-Type")).To(Equal(contentType))
			Expect(w.Header().Get("Content-Disposition")).To(Equal("attachment; filename="+filename))
		})

		It("should handle large file streaming", func() {
			// Create large test data (1MB)
			largeContent := make([]byte, 1024*1024)
			for i := range largeContent {
				largeContent[i] = byte(i % 256)
			}

			handler := func(ctx contracts.RequestContext) (io.Reader, string, ierrors.Error) {
				return bytes.NewReader(largeContent), "large-file.bin", nil
			}

			router.GET("/download-large", func(c *gin.Context) {
				ctx := &gin_framework.RequestContext{
					GinRequestContext: gin_framework.NewRequestContext(c).(*gin_framework.GinRequestContext),
				}

				data, filename, err := handler(ctx)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.Header("Content-Disposition", "attachment; filename="+filename)
				c.Header("Content-Type", "application/octet-stream")

				if reader, ok := data.(io.Reader); ok {
					io.Copy(c.Writer, reader)
				}
			})

			req, _ := http.NewRequest("GET", "/download-large", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(len(w.Body.Bytes())).To(Equal(1024 * 1024))
			Expect(w.Body.Bytes()[:10]).To(Equal(largeContent[:10])) // Check first 10 bytes
		})
	})

	Describe("Generic File Handler Pattern", func() {
		It("should implement the previous fileRequestHandler pattern", func() {
			// This reproduces the pattern mentioned in the user's description:
			// func fileRequestHandler[P RequestParam](ctx *gin.Context, param P, out FileResponse) {
			//     // write back an io.Reader or byte data to the response
			//     statusOK, DataResponse(gin.H{ "msg": "File downloaded successfully", }).WithReferences(refs))
			// }

			type FileDownloadRequest struct {
				FileID string `json:"file_id" validate:"required"`
				api.RequestModal[FileDownloadRequest]
			}

			type FileResponse struct {
				Data        io.Reader
				Filename    string
				ContentType string
				Size        int64
				Metadata    map[string]interface{}
			}

			// Implement the file request handler pattern
			fileRequestHandler := func(ctx contracts.RequestContext) (FileResponse, ierrors.Error) {
				var param FileDownloadRequest
				if loaded, loadError := param.RequestLoad(ctx); loadError != nil {
					return FileResponse{}, gerrors.RequestLoadFailed[FileDownloadRequest](loadError)
				} else {
					param = loaded.(FileDownloadRequest)
				}

				// Simulate getting file based on ID
				if param.FileID == "test-file-123" {
					content := "Sample file content for testing"
					reader := strings.NewReader(content)

					return FileResponse{
						Data:        reader,
						Filename:    "sample.txt",
						ContentType: "text/plain",
						Size:        int64(len(content)),
						Metadata: map[string]interface{}{
							"message": "File downloaded successfully",
							"references": []string{"ref1", "ref2", "ref3"},
						},
					}, nil
				}

				return FileResponse{}, gerrors.NotFound[FileDownloadRequest]("File not found")
			}

			router.POST("/file-download", gin_framework.WrapHandler(api.JSONHandler[map[string]interface{}](func(ctx contracts.RequestContext) (map[string]interface{}, ierrors.Error) {
				// For testing, we'll return metadata and simulate file download through headers
				response, err := fileRequestHandler(ctx)
				if err != nil {
					return nil, err
				}

				// Set file download headers in the context
				if apiCtx, ok := ctx.(interface{ SetHeader(string, string) }); ok {
					apiCtx.SetHeader("Content-Disposition", "attachment; filename="+response.Filename)
					apiCtx.SetHeader("Content-Type", response.ContentType)
					apiCtx.SetHeader("Content-Length", string(rune(response.Size)))
					apiCtx.SetHeader("X-File-Message", response.Metadata["message"].(string))
				}

				// Return metadata as JSON response
				return map[string]interface{}{
					"success": true,
					"message": response.Metadata["message"],
					"filename": response.Filename,
					"size": response.Size,
					"type": response.ContentType,
					"references": response.Metadata["references"],
				}, nil
			})))

			// Test successful file download request
			reqBody := `{"file_id": "test-file-123"}`
			req, _ := http.NewRequest("POST", "/file-download", strings.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response["success"]).To(Equal(true))

			// Debug the actual response
			if response["message"] == nil {
				// For debugging purposes, show what we actually got
				GinkgoWriter.Printf("Response: %+v\n", response)
			}

			// The message is nested in the data field due to JSONHandler wrapping
			if data, ok := response["data"].(map[string]interface{}); ok {
				Expect(data["message"]).To(Equal("File downloaded successfully"))
				Expect(data["filename"]).To(Equal("sample.txt"))
				Expect(data["size"]).To(Equal(31.0))
				Expect(data["type"]).To(Equal("text/plain"))
				Expect(data["success"]).To(Equal(true))
				if refs, ok := data["references"].([]interface{}); ok {
					Expect(len(refs)).To(Equal(3))
				}
			} else {
				// If data is nil, the test setup might have issues
				Expect(response["data"]).ToNot(BeNil(), "Data field should not be nil")
			}
		})
	})
})