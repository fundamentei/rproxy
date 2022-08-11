package rproxy

import "github.com/BurntSushi/toml"

// Config is for representing all the "configurable"s
type Config struct {
	General  general      `toml:"general"`
	Limits   limits       `toml:"limits"`
	Timeouts timeouts     `toml:"timeouts"`
	CORS     *corsOptions `toml:"cors"`
}

type general struct {
	// This should be defined if you care about full encryption. Since this proxy is known for the use of `Authorization`
	// header to encrypt the traffic, by adding a salt that's only known between the proxy and the client, we can ensure
	// that nothing will leak
	SharedKeySalt        string   `toml:"sharedKeySalt"`
	IsEncryptedHeaderKey string   `toml:"isEncryptedHeaderKey"`
	AllowedHosts         []string `toml:"allowedHosts"`
	DisallowedHosts      []string `toml:"disallowedHosts"`
	AllowedMethods       []string `toml:"allowedMethods"`
	// If enabled it won't pass through CORS requests. Not implemented yet
	UnsafeCORS bool `toml:"unsafeCORS"`
	// Is the address that the proxy will listen to when running locally
	Listen string `toml:"listen"`
}

// https://github.com/rs/cors/blob/master/cors.go#L32
type corsOptions struct {
	AllowedOrigins   []string `toml:"allowedOrigins"`
	AllowedMethods   []string `toml:"allowedMethods"`
	AllowedHeaders   []string `toml:"allowedHeaders"`
	ExposedHeaders   []string `toml:"exposedHeaders"`
	MaxAge           int      `toml:"maxAge"`
	AllowCredentials bool     `toml:"allowCredentials"`
	// AllowPrivateNetwork bool `toml:"allowPrivateNetwork"`
}

type limits struct {
	MaxRequestSizeInKB    uint64 `toml:"maxRequestSizeInKb"`
	MaxResponseSizeInKB   uint64 `toml:"maxResponseSizeInKb"`
	MaxIdleConns          int    `toml:"maxIdleConns"`
	MaxIdleConnsPerHost   int    `toml:"maxIdleConnsPerHost"`
	MaxConnsPerHost       int    `toml:"maxConnsPerHost"`
	MaxResponseHeaderInKB int64  `toml:"maxResponseHeaderInKb"`
}

// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
// https://i.stack.imgur.com/OWegJ.png
type timeouts struct {
	ClientTimeout uint32 `toml:"clientTimeout"`
	// DialerTimeoutMS limits the time spent establishing a TCP connection (if a new one is needed)
	DialerTimeout uint32 `toml:"dialerTimeout"`
	// TLSHandshakeTimeoutMS limits the time spent performing the TLS handshake
	TLSHandshakeTimeout uint32 `toml:"tlsHandshakeTimeout"`
	// ResponseHeaderTimeoutMS limits the time spent reading the headers of the response
	ResponseHeaderTimeout uint32 `toml:"responseHeaderTimeout"`
	// ExpectContinueTimeoutMS limits the time the client will wait between sending the request headers when including an
	// Expect: 100-continue and receiving the go-ahead to send the body
	ExpectContinueTimeout uint32 `toml:"expectContinueTimeout"`
	// IdleConnTimeoutMS limits the amount of time an idle connection is kept in the connection pool
	IdleConnTimeout uint32 `toml:"idleConnTimeout"`
}

// NewConfigFromFile is for parsing the configuration from the specified file
func NewConfigFromFile(filepath string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(filepath, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
