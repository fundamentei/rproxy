package rproxy

import (
	"log"
	"net/http"
	"time"
)

// dodgeFaviconRequest is for skipping "favicon.ico" requests when the proxy is open via browser for debugging purposes
func dodgeFaviconRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/favicon.ico" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// logIncomingRequest is for creating a handler func that logs all incoming requests
func logIncomingRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		lrw := &logResponseWriter{rw: w}
		handler.ServeHTTP(lrw, r)

		// Grab the URL we're proxying to
		parsedProxyToURL, err := requestURIToProxyURL(r.RequestURI)
		logProxyToURL := ""
		if parsedProxyToURL != nil && err == nil {
			logProxyToURL = parsedProxyToURL.String()
		}
		if logProxyToURL == "" {
			logProxyToURL = "-"
		}

		log.Printf("HTTP - %s - - %s \"%s %s %s\" %d %d %s %dus\n",
			realIP(r),
			now.Format("02/Jan/2006:15:04:05 -0700"),
			r.Method,
			logProxyToURL,
			r.Proto,
			lrw.Status(),
			lrw.Size(),
			r.UserAgent(),
			time.Since(now),
		)
	})
}
