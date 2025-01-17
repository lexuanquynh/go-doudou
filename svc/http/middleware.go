package ddhttp

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"github.com/ascarter/requestid"
	"github.com/felixge/httpsnoop"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/slok/goresilience"
	"github.com/slok/goresilience/bulkhead"
	"github.com/unionj-cloud/go-doudou/stringutils"
	"github.com/unionj-cloud/go-doudou/svc/config"
	"github.com/unionj-cloud/go-doudou/svc/logger"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// Metrics logs some metrics for http request
func Metrics(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(inner, w, r)
		logger.WithFields(logrus.Fields{
			"remoteAddr": r.RemoteAddr,
			"httpMethod": r.Method,
			"requestUri": r.URL.RequestURI(),
			"requestUrl": r.URL.String(),
			"statusCode": m.Code,
			"written":    m.Written,
			"duration":   m.Duration.String(),
		}).Info(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%s\n",
			r.RemoteAddr,
			r.Method,
			r.URL,
			m.Code,
			m.Written,
			m.Duration.String()))
	})
}

// borrowed from httputil unexported function drainBody
func copyReqBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func copyRespBody(b *bytes.Buffer) (b1, b2 *bytes.Buffer, err error) {
	if b == nil {
		return
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	return &buf, bytes.NewBuffer(buf.Bytes()), nil
}

func jsonMarshalIndent(data interface{}, prefix, indent string, disableHTMLEscape bool) (string, error) {
	b := &bytes.Buffer{}
	encoder := json.NewEncoder(b)
	encoder.SetEscapeHTML(!disableHTMLEscape)
	encoder.SetIndent(prefix, indent)
	if err := encoder.Encode(data); err != nil {
		return "", errors.Errorf("failed to marshal data to JSON, %s", err)
	}
	return b.String(), nil
}

func getReqBody(cp io.ReadCloser, r *http.Request) string {
	var contentType string
	if len(r.Header["Content-Type"]) > 0 {
		contentType = r.Header["Content-Type"][0]
	}
	var reqBody string
	if cp != nil {
		if strings.Contains(contentType, "multipart/form-data") {
			r.Body = cp
			if err := r.ParseMultipartForm(32 << 20); err == nil {
				reqBody = r.Form.Encode()
				if unescape, err := url.QueryUnescape(reqBody); err == nil {
					reqBody = unescape
				}
			} else {
				logger.Debug("call r.ParseMultipartForm(32 << 20) error: ", err)
			}
		} else if strings.Contains(contentType, "application/json") {
			data := make(map[string]interface{})
			if err := json.NewDecoder(cp).Decode(&data); err == nil {
				b, _ := json.MarshalIndent(data, "", "    ")
				reqBody = string(b)
			} else {
				logger.Debug("call json.NewDecoder(reqBodyCopy).Decode(&data) error: ", err)
			}
		} else {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(cp); err == nil {
				data := []rune(buf.String())
				end := len(data)
				if end > 1000 {
					end = 1000
				}
				reqBody = string(data[:end])
				if strings.Contains(contentType, "application/x-www-form-urlencoded") {
					if unescape, err := url.QueryUnescape(reqBody); err == nil {
						reqBody = unescape
					}
				}
			} else {
				logger.Debug("call buf.ReadFrom(reqBodyCopy) error: ", err)
			}
		}
	}
	return reqBody
}

func getRespBody(rec *httptest.ResponseRecorder) string {
	var (
		respBody string
		err      error
	)
	if strings.Contains(rec.Result().Header.Get("Content-Type"), "application/json") {
		var respBodyCopy *bytes.Buffer
		if respBodyCopy, rec.Body, err = copyRespBody(rec.Body); err == nil {
			data := make(map[string]interface{})
			if err := json.NewDecoder(rec.Body).Decode(&data); err == nil {
				b, _ := json.MarshalIndent(data, "", "    ")
				respBody = string(b)
			} else {
				logger.Debug("call json.NewDecoder(rec.Body).Decode(&data) error: ", err)
			}
		} else {
			logger.Debug("call respBodyCopy.ReadFrom(rec.Body) error: ", err)
		}
		rec.Body = respBodyCopy
	} else {
		data := []rune(rec.Body.String())
		end := len(data)
		if end > 1000 {
			end = 1000
		}
		respBody = string(data[:end])
	}
	return respBody
}

// Logger logs http request body and response body for debugging
func Logger(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RequestURI(), "/go-doudou/") || os.Getenv("GDD_LOG_LEVEL") != "debug" {
			inner.ServeHTTP(w, r)
			return
		}
		var (
			reqBodyCopy io.ReadCloser
			err         error
		)
		if reqBodyCopy, r.Body, err = copyReqBody(r.Body); err != nil {
			logger.Debug("call copyReqBody(r.Body) error: ", err)
		}

		rec := httptest.NewRecorder()
		inner.ServeHTTP(rec, r)

		reqBody := getReqBody(reqBodyCopy, r)
		start := time.Now()
		rid, _ := requestid.FromContext(r.Context())
		span := opentracing.SpanFromContext(r.Context())
		respBody := getRespBody(rec)
		reqQuery := r.URL.RawQuery
		if unescape, err := url.QueryUnescape(reqQuery); err == nil {
			reqQuery = unescape
		}
		fields := logrus.Fields{
			"remoteAddr":        r.RemoteAddr,
			"httpMethod":        r.Method,
			"requestUri":        r.URL.RequestURI(),
			"requestUrl":        r.URL.String(),
			"proto":             r.Proto,
			"host":              r.Host,
			"reqContentLength":  r.ContentLength,
			"reqHeader":         r.Header,
			"requestId":         rid,
			"reqQuery":          reqQuery,
			"reqBody":           reqBody,
			"respBody":          respBody,
			"statusCode":        rec.Result().StatusCode,
			"respHeader":        rec.Result().Header,
			"respContentLength": rec.Body.Len(),
			"elapsedTime":       time.Since(start).String(),
			"elapsed":           time.Since(start).Milliseconds(),
			"span":              fmt.Sprint(span),
		}
		var log string
		if log, err = jsonMarshalIndent(fields, "", "    ", true); err != nil {
			log = fmt.Sprintf("call jsonMarshalIndent(fields, \"\", \"    \", true) error: %s", err)
		}
		logger.WithFields(fields).Debugln(log)

		header := rec.Result().Header
		for k, v := range header {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Result().StatusCode)
		rec.Body.WriteTo(w)
	})
}

// Rest set Content-Type to application/json
func Rest(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stringutils.IsEmpty(w.Header().Get("Content-Type")) {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		}
		inner.ServeHTTP(w, r)
	})
}

// BasicAuth adds http basic auth validation
func BasicAuth(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := config.GddManageUser.Load()
		password := config.GddManagePass.Load()
		if stringutils.IsNotEmpty(username) || stringutils.IsNotEmpty(password) {
			user, pass, ok := r.BasicAuth()

			if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Provide user name and password"`)
				w.WriteHeader(401)
				w.Write([]byte("Unauthorised.\n"))
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
}

// Recover handles panic from processing incoming http request
func Recover(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				statusCode := http.StatusInternalServerError
				if err, ok := e.(error); ok {
					if errors.Is(err, context.Canceled) {
						statusCode = http.StatusBadRequest
					}
				}
				logger.Errorf("panic: %+v\n\nstacktrace from panic: %s\n", e, string(debug.Stack()))
				http.Error(w, fmt.Sprintf("%v", e), statusCode)
			}
		}()
		inner.ServeHTTP(w, r)
	})
}

// Tracing add jaeger tracing middleware
func Tracing(inner http.Handler) http.Handler {
	return nethttp.Middleware(
		opentracing.GlobalTracer(),
		inner,
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method + " " + r.URL.Path
		}))
}

// BulkHead add bulk head pattern middleware based on https://github.com/slok/goresilience
// workers is the number of workers in the execution pool.
// maxWaitTime is the max time an incoming request will wait to execute before being dropped its execution and return 429 response.
func BulkHead(workers int, maxWaitTime time.Duration) func(inner http.Handler) http.Handler {
	runner := goresilience.RunnerChain(
		bulkhead.NewMiddleware(bulkhead.Config{
			Workers:     workers,
			MaxWaitTime: maxWaitTime,
		}),
	)
	return func(inner http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := runner.Run(r.Context(), func(_ context.Context) error {
				inner.ServeHTTP(w, r)
				return nil
			})
			if err != nil {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
			}
		})
	}
}
