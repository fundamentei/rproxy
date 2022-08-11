package rproxy

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gobwas/glob"
	"github.com/samber/lo"
)

type handler struct {
	// General
	sharedKeySalt        string
	isEncryptedHeaderKey string

	allowedMethods  []string
	allowedHosts    []string
	disallowedHosts []string

	// Limits
	maxRequestSizeInKb  uint64
	maxResponseSizeInKb uint64

	httpClient *http.Client
}

// NewHandler is for creating a new handler
func NewHandler(cfg *Config) http.Handler {
	return gziphandler.GzipHandler(dodgeFaviconRequest(logIncomingRequest(&handler{
		sharedKeySalt:        cfg.General.SharedKeySalt,
		isEncryptedHeaderKey: cfg.General.IsEncryptedHeaderKey,

		allowedMethods:  cfg.General.AllowedMethods,
		allowedHosts:    cfg.General.AllowedHosts,
		disallowedHosts: cfg.General.DisallowedHosts,

		maxRequestSizeInKb:  cfg.Limits.MaxRequestSizeInKB,
		maxResponseSizeInKb: cfg.Limits.MaxResponseSizeInKB,

		httpClient: makeClientFromConfig(cfg),
	})))
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(h.isEncryptedHeaderKey, "false")
	// Verify if the method we're requesting the destination with is allowed
	if !lo.Contains(h.allowedMethods, r.Method) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Parse the incoming URL being proxied
	proxyToURL, err := requestURIToProxyURL(r.RequestURI)
	if proxyToURL == nil || err != nil || proxyToURL.Scheme == "" || proxyToURL.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Verify if the "Host" we're proxying to is blacklisted. This is primarly useful to avoid recursive proxying
	if h.isHostInGlobList(h.disallowedHosts, proxyToURL.Host) {
		log.Printf("Denying request to Host: %q", proxyToURL.Host)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// Verify if the "Host" we're proxying to is whitelisted
	if !h.isHostInGlobList(h.allowedHosts, proxyToURL.Host) {
		log.Printf("Denying request to Host: %q", proxyToURL.Host)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Rebuilds from scratch the URL we're proxying to
	preq, err := http.NewRequest(r.Method, fmt.Sprintf("%s://%s", proxyToURL.Scheme, proxyToURL.Host), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// http: Request.RequestURI can't be set in client requests
	// https://go.dev/src/net/http/client.go
	preq.RequestURI = ""
	h.copyHeaders(preq.Header, r.Header)
	h.delHopHeaders(preq.Header)

	pres, err := h.httpClient.Do(preq)
	// Provide context information for logging
	logDetailsLine := fmt.Sprintf(
		"%s %s %q %s %q",
		realIP(r),
		r.Method,
		proxyToURL,
		r.Proto,
		r.UserAgent(),
	)

	if err != nil {
		if pres != nil {
			w.WriteHeader(pres.StatusCode)
			log.Printf("Couldn't execute the request: %s %v", logDetailsLine, err)
			return
		}
		// Since we can't determine the status code, we'll just return a 500
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Couldn't execute the request: %s %v", logDetailsLine, err)
		return
	}

	// Provide context information for logging
	logDetailsLine = fmt.Sprintf(
		"%s %s %q %s %s %q",
		realIP(r),
		r.Method,
		proxyToURL,
		r.Proto,
		pres.Status,
		r.UserAgent(),
	)

	defer pres.Body.Close()
	brd := io.LimitReader(pres.Body, int64(h.maxResponseSizeInKb*1024))
	if pres.Header.Get(http.CanonicalHeaderKey(hContentEncoding)) == "gzip" {
		if gzr, err := gzip.NewReader(brd); gzr != nil && err == nil {
			defer gzr.Close()
			brd = gzr
		}
	}
	body, err := ioutil.ReadAll(brd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authorization := pres.Header.Get(hAuthorization)
	key := aesKey(authorization + h.sharedKeySalt)
	erb, err := aesEncrypt(key, body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Couldn't encrypt response: %s %v", logDetailsLine, err)
		return
	}

	// We're ready to start transfering the encrypted response
	w.Header().Del(hContentLength)
	w.Header().Set(hContentLength, strconv.Itoa(len(erb)))
	w.Header().Set(h.isEncryptedHeaderKey, "true")
	w.WriteHeader(pres.StatusCode)
	w.Write(erb)
	return
}

// Hop-by-hop headers. These are removed when sent to the backend
// See: http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // Canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func (h *handler) isHostInGlobList(hosts []string, candidate string) bool {
	for _, host := range hosts {
		g := glob.MustCompile(host)
		if g.Match(candidate) {
			return true
		}
	}
	return false
}

func (h *handler) delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func (h *handler) copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func makeClientFromConfig(cfg *Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   time.Duration(cfg.Timeouts.DialerTimeout * uint32(time.Second)),
				KeepAlive: 30 * time.Second,
			}).Dial,
			ForceAttemptHTTP2:      true,
			MaxIdleConns:           cfg.Limits.MaxIdleConns,
			MaxIdleConnsPerHost:    cfg.Limits.MaxIdleConnsPerHost,
			MaxConnsPerHost:        cfg.Limits.MaxConnsPerHost,
			MaxResponseHeaderBytes: int64(cfg.Limits.MaxResponseHeaderInKB * 1024),
			TLSHandshakeTimeout:    time.Duration(cfg.Timeouts.TLSHandshakeTimeout * uint32(time.Second)),
			ResponseHeaderTimeout:  time.Duration(cfg.Timeouts.ResponseHeaderTimeout * uint32(time.Second)),
			ExpectContinueTimeout:  time.Duration(cfg.Timeouts.ExpectContinueTimeout * uint32(time.Second)),
			IdleConnTimeout:        time.Duration(cfg.Timeouts.IdleConnTimeout * uint32(time.Second)),
			DisableCompression:     true,
		},
	}
}
