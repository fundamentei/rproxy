package rproxy

import (
	"net"
	"net/http"
	"time"
)

type handler struct {
	httpClient *http.Client
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func makeClientFromConfig(cfg *Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   time.Duration(cfg.Timeouts.DialerTimeoutMS * uint32(time.Microsecond)),
				KeepAlive: 30 * time.Second,
			}).Dial,
			ForceAttemptHTTP2:      true,
			MaxIdleConns:           cfg.Limits.maxIdleConns,
			MaxIdleConnsPerHost:    cfg.Limits.maxIdleConnsPerHost,
			MaxConnsPerHost:        cfg.Limits.maxConnsPerHost,
			MaxResponseHeaderBytes: int64(cfg.Limits.maxResponseHeaderInKB * 1024),
			TLSHandshakeTimeout:    time.Duration(cfg.Timeouts.TLSHandshakeTimeoutMS * uint32(time.Microsecond)),
			ResponseHeaderTimeout:  time.Duration(cfg.Timeouts.ResponseHeaderTimeoutMS * uint32(time.Microsecond)),
			ExpectContinueTimeout:  time.Duration(cfg.Timeouts.ExpectContinueTimeoutMS * uint32(time.Microsecond)),
			IdleConnTimeout:        time.Duration(cfg.Timeouts.IdleConnTimeoutMS * uint32(time.Microsecond)),
		},
	}
}

// NewHandler is for creating a new handler
func NewHandler(cfg *Config) http.Handler {
	return dodgeFaviconRequest(logIncomingRequest(&handler{
		httpClient: makeClientFromConfig(cfg),
	}))
}
