package rproxy

import (
	"net/http"
	"sync"
)

type logResponseWriter struct {
	rw              http.ResponseWriter
	writeHeaderOnce sync.Once
	status          int
	size            int
}

func (l *logResponseWriter) Header() http.Header {
	return l.rw.Header()
}

func (l *logResponseWriter) WriteHeader(statusCode int) {
	l.writeHeaderOnce.Do(func() {
		l.status = statusCode
		l.rw.WriteHeader(statusCode)
	})
}

func (l *logResponseWriter) Write(data []byte) (int, error) {
	l.writeHeaderOnce.Do(func() {
		if l.status == 0 {
			l.status = 200
		}
	})
	l.size = len(data)
	return l.rw.Write(data)
}

func (l *logResponseWriter) Status() int {
	return l.status
}
func (l *logResponseWriter) Size() int {
	return l.size
}
