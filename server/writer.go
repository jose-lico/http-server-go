package server

import (
	"bytes"
	"net/http"
)

// Implement http.ResponseWriter
type Writer struct {
	headers http.Header
	body    *bytes.Buffer
	status  int
}

func NewWriter() *Writer {
	return &Writer{headers: make(http.Header), body: &bytes.Buffer{}}
}

func (rw *Writer) Header() http.Header {
	return rw.headers
}

func (rw *Writer) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}

func (rw *Writer) WriteHeader(statusCode int) {
	rw.status = statusCode
}
