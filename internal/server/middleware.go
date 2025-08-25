package server

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"

	"project/internal/domain/errors"

	"github.com/gin-gonic/gin"
)

type dualCloser struct {
	io.Reader
	gzipReader io.Closer
	bodyCloser io.Closer
}

func (dc *dualCloser) Close() error {
	var err1, err2 error
	if dc.gzipReader != nil {
		err1 = dc.gzipReader.Close()
	}
	if dc.bodyCloser != nil {
		err2 = dc.bodyCloser.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func GzipRequestDecompress() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		encoding := strings.ToLower(ctx.GetHeader("Content-Encoding"))
		if strings.Contains(encoding, "gzip") {
			gr, err := gzip.NewReader(ctx.Request.Body)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": errors.ErrInvalidGzipRequest.Error()})
				return
			}

			ctx.Request.Body = &dualCloser{
				Reader:     gr,
				gzipReader: gr,
				bodyCloser: ctx.Request.Body,
			}

			ctx.Request.Header.Del("Content-Encoding")
			ctx.Request.Header.Del("Content-Length")
		}
		ctx.Next()
	}
}

type gzipResponseWriter struct {
	writer      gin.ResponseWriter
	gw          *gzip.Writer
	gzipEnabled bool
	statusCode  int
	totalSize   int
	preBuf      bytes.Buffer
}

const minCompressSize = 1024

var nonCompressibleStatuses = map[int]bool{
	http.StatusNoContent:         true,
	http.StatusNotModified:       true,
	http.StatusPartialContent:    true,
	http.StatusMultipleChoices:   true,
	http.StatusMovedPermanently:  true,
	http.StatusFound:             true,
	http.StatusSeeOther:          true,
	http.StatusTemporaryRedirect: true,
	http.StatusPermanentRedirect: true,
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if w.gzipEnabled {
		n, err := w.gw.Write(data)
		if err != nil {
			return n, errors.ErrGzipCompressionFailed
		}
		w.totalSize += n
		return n, nil
	}

	startLen := w.preBuf.Len()
	if _, err := w.preBuf.Write(data); err != nil {
		return 0, err
	}
	w.totalSize += len(data)

	if w.preBuf.Len() >= minCompressSize && w.mayCompress() {
		w.enableGzip()
		if w.gw != nil {
			if _, err := w.gw.Write(w.preBuf.Bytes()); err != nil {
				return 0, errors.ErrGzipCompressionFailed
			}
			w.preBuf.Reset()
		}
	}

	return w.preBuf.Len() - startLen, nil
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func (w *gzipResponseWriter) mayCompress() bool {
	if nonCompressibleStatuses[w.statusCode] {
		return false
	}
	if w.writer.Header().Get("Content-Encoding") != "" {
		return false
	}
	ct := w.writer.Header().Get("Content-Type")
	return isCompressibleContentType(ct)
}

func (w *gzipResponseWriter) enableGzip() {
	w.writer.Header().Del("Content-Length")
	w.writer.Header().Set("Content-Encoding", "gzip")
	vary := w.writer.Header().Get("Vary")
	if vary == "" {
		w.writer.Header().Set("Vary", "Accept-Encoding")
	} else if !strings.Contains(vary, "Accept-Encoding") {
		w.writer.Header().Set("Vary", vary+", Accept-Encoding")
	}
	w.gw = gzip.NewWriter(w.writer)
	w.gzipEnabled = true
}

func (w *gzipResponseWriter) Flush() {
	if w.gw != nil {
		_ = w.gw.Flush()
	} else if w.preBuf.Len() > 0 {
		_, _ = w.writer.Write(w.preBuf.Bytes())
		w.preBuf.Reset()
	}
	w.writer.Flush()
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.writer.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) CloseNotify() <-chan bool { return w.writer.CloseNotify() }

func (w *gzipResponseWriter) Pusher() http.Pusher { return w.writer.Pusher() }

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.writer.Hijack() }

func (w *gzipResponseWriter) Size() int { return w.writer.Size() }

func (w *gzipResponseWriter) Status() int { return w.writer.Status() }

func (w *gzipResponseWriter) WriteHeaderNow() { w.writer.WriteHeaderNow() }

func (w *gzipResponseWriter) Written() bool { return w.writer.Written() }

func GzipResponseCompress() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method == http.MethodHead {
			ctx.Next()
			return
		}

		acceptEnc := strings.ToLower(ctx.GetHeader("Accept-Encoding"))
		if !strings.Contains(acceptEnc, "gzip") {
			ctx.Next()
			return
		}

		vary := ctx.Writer.Header().Get("Vary")
		if vary == "" {
			ctx.Writer.Header().Set("Vary", "Accept-Encoding")
		} else if !strings.Contains(vary, "Accept-Encoding") {
			ctx.Writer.Header().Set("Vary", vary+", Accept-Encoding")
		}

		gw := &gzipResponseWriter{writer: ctx.Writer}
		ctx.Writer = gw

		ctx.Next()

		if gw.gw != nil {
			if err := gw.gw.Close(); err != nil {
				_ = ctx.Error(errors.ErrGzipCompressionFailed)
			}
		} else if gw.preBuf.Len() > 0 {
			if _, err := gw.writer.Write(gw.preBuf.Bytes()); err != nil {
				_ = ctx.Error(err)
			}
			gw.preBuf.Reset()
		}
	}
}

func isCompressibleContentType(ct string) bool {
	if ct == "" {
		return false
	}

	lower := strings.ToLower(ct)
	if strings.HasPrefix(lower, "text/event-stream") {
		return false
	}

	compressiblePrefixes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"text/html",
		"text/css",
		"text/plain",
		"text/xml",
		"text/javascript",
	}

	for _, prefix := range compressiblePrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}
