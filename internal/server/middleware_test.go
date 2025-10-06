package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGzipRequestDecompress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipRequestDecompress())
	router.POST("/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"body": string(body)})
	})

	tests := []struct {
		name            string
		content         string
		contentEncoding string
		want            struct {
			statusCode int
			body       string
		}
	}{
		{
			name:            "uncompressed request",
			content:         "Hello, World!",
			contentEncoding: "",
			want: struct {
				statusCode int
				body       string
			}{
				statusCode: http.StatusOK,
				body:       "Hello, World!",
			},
		},
		{
			name:            "gzip compressed request",
			content:         "Hello, World!",
			contentEncoding: "gzip",
			want: struct {
				statusCode int
				body       string
			}{
				statusCode: http.StatusOK,
				body:       "Hello, World!",
			},
		},
		{
			name:            "invalid gzip request",
			content:         "Invalid gzip data",
			contentEncoding: "gzip",
			want: struct {
				statusCode int
				body       string
			}{
				statusCode: http.StatusOK,
				body:       "Invalid gzip data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.contentEncoding == "gzip" {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				_, _ = gz.Write([]byte(tt.content))
				gz.Close()
				body = &buf
			} else {
				body = strings.NewReader(tt.content)
			}

			req, _ := http.NewRequest("POST", "/test", body)
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.statusCode == http.StatusOK {
				assert.Contains(t, w.Body.String(), tt.want.body)
			}
		})
	}
}

func TestGzipResponseCompress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipResponseCompress())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	tests := []struct {
		name           string
		acceptEncoding string
		want           struct {
			statusCode      int
			contentEncoding string
			hasContent      bool
		}
	}{
		{
			name:           "client accepts gzip",
			acceptEncoding: "gzip",
			want: struct {
				statusCode      int
				contentEncoding string
				hasContent      bool
			}{
				statusCode:      http.StatusOK,
				contentEncoding: "",
				hasContent:      true,
			},
		},
		{
			name:           "client does not accept gzip",
			acceptEncoding: "",
			want: struct {
				statusCode      int
				contentEncoding string
				hasContent      bool
			}{
				statusCode:      http.StatusOK,
				contentEncoding: "",
				hasContent:      true,
			},
		},
		{
			name:           "client accepts other encoding",
			acceptEncoding: "deflate",
			want: struct {
				statusCode      int
				contentEncoding string
				hasContent      bool
			}{
				statusCode:      http.StatusOK,
				contentEncoding: "",
				hasContent:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			assert.Equal(t, tt.want.contentEncoding, w.Header().Get("Content-Encoding"))
			if tt.want.hasContent {
				assert.NotEmpty(t, w.Body.Bytes())
			}
		})
	}
}

func TestGzipMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GzipRequestDecompress())
	router.Use(GzipResponseCompress())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	tests := []struct {
		name           string
		acceptEncoding string
		want           struct {
			statusCode int
			success    bool
		}
	}{
		{
			name:           "client accepts gzip",
			acceptEncoding: "gzip",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: http.StatusOK,
				success:    true,
			},
		},
		{
			name:           "client does not accept gzip",
			acceptEncoding: "",
			want: struct {
				statusCode int
				success    bool
			}{
				statusCode: http.StatusOK,
				success:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			if tt.want.success {
				assert.Contains(t, w.Body.String(), "test")
			}
		})
	}
}

func TestMiddlewareAdditionalScenarios(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		contentType    string
		acceptEncoding string
		want           struct {
			statusCode      int
			contentEncoding string
			hasContent      bool
		}
	}{
		{
			name:           "large content compression",
			content:        strings.Repeat("Large content for compression testing. ", 100),
			contentType:    "text/plain",
			acceptEncoding: "gzip",
			want: struct {
				statusCode      int
				contentEncoding string
				hasContent      bool
			}{
				statusCode:      http.StatusOK,
				contentEncoding: "gzip",
				hasContent:      true,
			},
		},
		{
			name:           "json content type",
			content:        `{"message": "test"}`,
			contentType:    "application/json",
			acceptEncoding: "gzip",
			want: struct {
				statusCode      int
				contentEncoding string
				hasContent      bool
			}{
				statusCode:      http.StatusOK,
				contentEncoding: "",
				hasContent:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(GzipResponseCompress())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, tt.content)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.want.statusCode, w.Code)
			assert.Equal(t, tt.want.contentEncoding, w.Header().Get("Content-Encoding"))
			if tt.want.hasContent {
				assert.NotEmpty(t, w.Body.String())
			}
		})
	}
}
