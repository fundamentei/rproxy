package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gobwas/glob"
	"github.com/samber/lo"
)

var (
	// // https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html#configuration-envvars-runtime
	isRunningInLambda = strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda")
)

// rproxyHandler is a handler for the proxy that proxies requests and perform AES encryption over the response before
// sending it to the client. The API is as follows: https://rproxy.fundamentei.io/{URL}
type rproxyHandler struct {
	disallowedHosts          []string
	allowedHosts             []string
	allowedMethods           []string
	httpClient               *http.Client
	encryptionKeyFromRequest func(r *http.Request) string
}

func (rpxy *rproxyHandler) isHostInList(hosts []string, candidate string) bool {
	for _, host := range hosts {
		g := glob.MustCompile(host)
		if g.Match(candidate) {
			return true
		}
	}
	return false
}

func realIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func isValidURL(rawURL string) bool {
	_, err := url.Parse(rawURL)
	return err == nil
}

// requestURIToProxyURL is for fetching the destination URL from the request. It allows the following inputs:
// /https%3A%2F%2Fproduction.api-lambda.fundamentei.io
// /https://production.api-lambda.fundamentei.io
// /aHR0cHM6Ly9wcm9kdWN0aW9uLmFwaS1sYW1iZGEuZnVuZGFtZW50ZWkuaW8=
func requestURIToProxyURL(requestURI string) string {
	// Removes the leading slash from destination URL that may come up with the request
	if strings.HasPrefix(requestURI, "/") {
		requestURI = strings.TrimPrefix(requestURI, "/")
	}
	d, err := base64.RawStdEncoding.DecodeString(requestURI)
	if err != nil {
		decodedURI, err := url.QueryUnescape(requestURI)
		if err != nil {
			return ""
		}
		// URL to proxy to didn't come encoded in the request
		if !isValidURL(decodedURI) {
			return ""
		}
		return decodedURI
	}
	decodedRawURL := string(d)
	if !isValidURL(decodedRawURL) {
		return ""
	}
	return decodedRawURL
}

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

func logIncomingRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		lrw := &logResponseWriter{rw: w}
		handler.ServeHTTP(lrw, r)

		log.Printf("HTTP - %s - - %s \"%s %s %s\" %d %d %s %dus\n",
			realIP(r),
			now.Format("02/Jan/2006:15:04:05 -0700"),
			r.Method,
			requestURIToProxyURL(r.RequestURI),
			r.Proto,
			lrw.Status(),
			lrw.Size(),
			r.UserAgent(),
			time.Since(now),
		)
	})
}

var (
	xFndmIsEncrypted = http.CanonicalHeaderKey("X-Fndm-Is-Encrypted")
	hContentLength   = http.CanonicalHeaderKey("Content-Length")
	hAuthorization   = http.CanonicalHeaderKey("Authorization")
)

func (rpxy *rproxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: improve this by only allowing to proxy a specific list of hosts that comes from a config file
	// TODO: log all incoming data
	if !lo.Contains(rpxy.allowedMethods, r.Method) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// proxyToURL, err := url.Parse(r.URL.RawPath)
	// if err != nil {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	requestURI := requestURIToProxyURL(r.RequestURI)
	if requestURI == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	actualURL, err := url.Parse(requestURI)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Verify if the "Host" we're proxying to is blacklisted. This is primarly useful to avoid recursive proxying
	if rpxy.isHostInList(rpxy.disallowedHosts, actualURL.Host) {
		log.Printf("Denying request to Host: %q", actualURL.Host)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// Verify if the "Host" we're proxying to is whitelisted
	if !rpxy.isHostInList(rpxy.allowedHosts, actualURL.Host) {
		log.Printf("Denying request to Host: %q", actualURL.Host)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	req, err := http.NewRequest(r.Method, fmt.Sprintf("%s://%s", actualURL.Scheme, actualURL.Host), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	req.Host = actualURL.Host
	r.URL.Opaque = actualURL.RequestURI()
	rpxy.delHopHeaders(r.Header)
	rpxy.transferHeaders(req.Header, r.Header)

	res, err := rpxy.httpClient.Do(req)
	if err != nil {
		if res != nil {
			w.WriteHeader(res.StatusCode)
			return
		}
		// Since we can't determine the status code, we'll just return a 500
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer res.Body.Close()

	// Proxy ("copy") everything that came back from the destination response
	rpxy.delHopHeaders(res.Header)
	rpxy.transferHeaders(w.Header(), res.Header)

	// Provide context information for logging
	logDetailsLine := fmt.Sprintf(
		"%s %s %q %s %s %q",
		realIP(r),
		r.Method,
		requestURIToProxyURL(r.RequestURI),
		r.Proto,
		res.Status,
		r.UserAgent(),
	)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Couldn't read response body: %s %v", logDetailsLine, err)
		return
	}

	encryptionKey := rpxy.encryptionKeyFromRequest(r)
	// If no encryption key was provided for request then we can't encrypt the response
	if encryptionKey == "" {
		w.WriteHeader(res.StatusCode)
		// Indicates
		w.Header().Set(xFndmIsEncrypted, "false")
		io.Copy(w, res.Body)
		return
	}
	// Indicates that the response is encrypted
	w.Header().Set(xFndmIsEncrypted, "true")
	safeAESKey := aesKey(encryptionKey)
	encrypted, err := aesEncrypt([]byte(safeAESKey), body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Couldn't encrypt response: %s %v", logDetailsLine, err)
		return
	}

	// We're ready to start transfering the encrypted response
	w.Header().Del(hContentLength)
	w.Header().Set(hContentLength, strconv.Itoa(len(encrypted)))

	if _, err := io.Copy(w, bytes.NewReader(encrypted)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Couldn't transfer encrypted response to writer: %s %v", logDetailsLine, err)
		return
	}
	w.WriteHeader(res.StatusCode)
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

func (rpxy *rproxyHandler) delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func (rpxy *rproxyHandler) transferHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func run() error {
	rpxyHandler := logIncomingRequest(&rproxyHandler{
		disallowedHosts: []string{"rproxy.fundamentei.io", "rproxy.fndm.to"},
		allowedHosts:    []string{"*.fundamentei.io", "*.fundamentei.com", "*.fndm.to", "localhost:*", "localhost"},
		allowedMethods:  []string{http.MethodGet, http.MethodPost},
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		encryptionKeyFromRequest: func(r *http.Request) string {
			authorization := r.Header.Get(hAuthorization)
			parts := strings.SplitN(authorization, " ", 2)
			// Need to handle the case where the authorization header is not present
			if len(parts) != 2 {
				return ""
			}
			// Handle non-JWT authorization headers
			if strings.ToLower(parts[0]) != "bearer" {
				return ""
			}
			return parts[1]
		},
	})

	if isRunningInLambda {
		lambda.Start(httpadapter.NewV2(rpxyHandler).ProxyWithContext)
		return nil
	}

	return http.ListenAndServe(":25259", rpxyHandler)
}

func main() {
	msg, _ := aesEncrypt(aesKey("foo123"), []byte("hello world"))
	fmt.Println(msg)

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// This will usually receive a JWT token as an input, but since the token has more than 32 bytes, we'll hash it so it
// can be used as a key for AES encryption
func aesKey(input string) []byte {
	h := md5.New()
	h.Write([]byte(input))
	key := hex.EncodeToString(h.Sum(nil))
	return []byte(key)
}

func aesEncrypt(key []byte, input []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	payload, _ := pkcs7Padding(input, aes.BlockSize)
	output := make([]byte, aes.BlockSize+len(payload))

	// CBC mode works on blocks, so plain text may need to be padded to the next whole block
	// https://www.rfc-editor.org/rfc/rfc5246#section-6.2.3.2
	iv := output[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(output[aes.BlockSize:], payload)

	return output, nil
}

func withPadding(payload []byte, blockSize int) []byte {
	if len(payload)%aes.BlockSize == 0 {
		return payload
	}
	data := make([]byte, int(len(payload)/blockSize+1)*blockSize)
	copy(data, payload)
	return data
}

// Right-pads the data string with 1 to n bytes according to PKCS#7 where n is the block size. The size of the result
// is x times n, where x is at least 1. The version of PKCS#7 padding used is the one defined in RFC 5652 chapter 6.3.
// This padding is identical to PKCS#5 padding for 8 byte block ciphers such as DES
func pkcs7Padding(payload []byte, blockSize int) ([]byte, uint8) {
	padding := blockSize - len(payload)%blockSize
	text := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(payload, text...), uint8(padding)
}

func zeroPad(payload []byte, blockSize int) []byte {
	padding := blockSize - len(payload)%blockSize
	text := bytes.Repeat([]byte{byte('0')}, padding)
	return append(payload, text...)
}
