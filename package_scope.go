package cbapiclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/newrelic/go-agent"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

// Defaults is a container for setting package level values
type Defaults struct {
	// ServiceName is the name of the calling service
	ServiceName string

	// NewRelicTransactionProviderFunc is a function that
	// provides the New Relic transaction to be used in the
	// HTTP Request.  If this function is not set, the client
	// will create a new New Relic transaction
	NewRelicTransactionProviderFunc func(ctx context.Context) (newrelic.Transaction, bool)

	// TracerProviderFunc is a function that provides
	// the opentracing.Tracer for tracing HTTP requests
	TracerProviderFunc func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span)

	// ContextLoggerProviderFunc is a function that provides
	// a logger from the current context.  If this function
	// is not set, the client will create a new logger for
	// the Request.
	// Deprecated: This function will return a generic Logger interface (defined in github.com/InVisionApp/go-logger) instead of a vendor-specific implementation
	ContextLoggerProviderFunc func(ctx context.Context) (*logrus.Entry, bool)

	// RequestIDProviderFunc is a function that provides the
	// parent Request id used in tracing the caller's Request.
	// If this function is not set, the client will generate
	// a new UUID for the Request id.
	RequestIDProviderFunc func(ctx context.Context) (string, bool)

	// RequestSourceProviderFunc is a function that provides
	// the Request-Source header
	RequestSourceProviderFunc func(ctx context.Context) (string, bool)

	// UserAgent is a package-level user agent header used for
	// each outgoing request
	UserAgent string

	// StrictREQ014 will cancel any request and return an error if any of the following
	// headers are missing:
	// 		Request-ID
	// 		Request-Source
	// 		Calling-Service
	StrictREQ014 bool

	// StatsdRate is the statsd reporting rate
	StatsdRate float64

	// StatsdSuccessTag is the tag added to the statsd metric when the request succeeds (200 <= status_code < 300)
	StatsdSuccessTag string

	// StatsdFailureTag is the tag added to the statsd metric when the request fails
	StatsdFailureTag string
}

var (
	pkgServiceName               string
	pkgUserAgent                 string
	pkgNRTxnProviderFunc         func(ctx context.Context) (newrelic.Transaction, bool)
	pkgTracerProviderFunc        func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span)
	pkgCtxLoggerProviderFunc     func(ctx context.Context) (*logrus.Entry, bool)
	pkgRequestIDProviderFunc     func(cxt context.Context) (string, bool)
	pkgRequestSourceProviderFunc func(cxt context.Context) (string, bool)
	pkgOnce                      sync.Once
	pkgStrictREQ014              bool
	pkgStatsdRate                float64
	pkgStatsdSuccessTag          string
	pkgStatsdFailureTag          string

	envHTTPMocking = "MOCKING_HTTP"
)

//
// Package Level Functions
// ========================================================
//

// ensurePackageVariables makes sure that the package level
// variables are set.  This function runs once, then no-ops
// on subsequent calls
func ensurePackageVariables() {
	pkgOnce.Do(func() {

		// we need something to be set as a service name
		if pkgServiceName == "" {
			// if caller didnt set it, look in env
			pkgServiceName = os.Getenv("SERVICE_NAME")
			if pkgServiceName == "" {
				// if not in env, just use the hostname
				pkgServiceName = os.Getenv("HOSTNAME")
			}
		}

		// user agent is service name + hostname
		if pkgUserAgent == "" {
			if pkgServiceName == os.Getenv("HOSTNAME") {
				pkgUserAgent = pkgServiceName
			} else {
				pkgUserAgent = fmt.Sprintf("%s-%s", pkgServiceName, os.Getenv("HOSTNAME"))
			}
		}

		// make sure new relic transaction provider exists
		if pkgNRTxnProviderFunc == nil {
			logrus.WithField("type", NAME).
				Warn("cbapiclient: no NewRelicTransactionProviderFunc set")
			pkgNRTxnProviderFunc = func(ctx context.Context) (newrelic.Transaction, bool) {
				// the newrelic StartSegment function will start a new transaction
				return nil, false
			}
		}

		// make sure the context logger provider exists
		if pkgCtxLoggerProviderFunc == nil {
			logrus.WithField("type", NAME).
				Warn("cbapiclient: No ContextLoggerProviderFunc default set.  A new logger will be " +
					"used for each request")
			pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
				return logrus.NewEntry(logrus.New()), true
			}
		}

		// make sure the Request id provider exists
		if pkgRequestIDProviderFunc == nil {
			logrus.WithField("type", NAME).
				Warn("cbapiclient: No RequestIDProviderFunc default set.  The Request-ID header will " +
					"be absent in each request unless set manually")
			pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
				return "", true
			}
		}

		// make sure the request source provider exists
		if pkgRequestSourceProviderFunc == nil {
			logrus.WithField("type", NAME).
				Warn("cbapiclient: No RequestSourceProviderFunc default set.  The Request-Source header " +
					"will be absent in each request unless set manually")
			pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
				return "", false
			}
		}

		// ensure statsd success and failure tags exist
		if pkgStatsdSuccessTag == "" {
			logrus.WithField("type", NAME).Info("cbapiclient: no statsd success tag provided.  using processed:success.")
			pkgStatsdSuccessTag = "processed:success"
		}

		if pkgStatsdFailureTag == "" {
			logrus.WithField("type", NAME).Info("cbapiclient: no statsd failure tag provided.  using processed:failure.")
			pkgStatsdFailureTag = "processed:failure"
		}
	})
}

// SetDefaults will apply package-level default values to
// be used on all requests
func SetDefaults(defaults *Defaults) {
	pkgServiceName = defaults.ServiceName
	pkgNRTxnProviderFunc = defaults.NewRelicTransactionProviderFunc
	pkgCtxLoggerProviderFunc = defaults.ContextLoggerProviderFunc
	pkgRequestIDProviderFunc = defaults.RequestIDProviderFunc
	pkgRequestSourceProviderFunc = defaults.RequestSourceProviderFunc
	pkgUserAgent = defaults.UserAgent
	pkgStrictREQ014 = defaults.StrictREQ014
	pkgStatsdRate = defaults.StatsdRate
	pkgStatsdSuccessTag = defaults.StatsdSuccessTag
	pkgStatsdFailureTag = defaults.StatsdFailureTag
	pkgTracerProviderFunc = defaults.TracerProviderFunc
}

// this creates a http client with sensible defaults
func newHTTPClient() *http.Client {
	// all http mocking libraries can override the default http client,
	// but many cannot override clients that have been tuned with custom
	// transports.  If this env var is set, we return the standard
	// http client.
	if os.Getenv(envHTTPMocking) != "" {
		return http.DefaultClient
	}

	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   sockTimeout,
				DualStack: true,
				KeepAlive: keepAlive,
			}).DialContext,
			MaxIdleConnsPerHost:   maxIdleConnsPerHost,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleTimeout,
			TLSHandshakeTimeout:   tlsTimeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	return client
}

// NewClient will initialize and return a new client with a
// request and endpoint.  The client's content type defaults
// to application/json
func NewClient(uri string) (*Client, error) {

	ensurePackageVariables()

	ep, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, err
	}

	c := &Client{
		endpoint: ep,
		method:   http.MethodGet,
		client:   newHTTPClient(),
		headers: map[string]string{
			userAgentHeader:      pkgUserAgent,
			contentTypeHeader:    jsonType,
			callingServiceHeader: pkgServiceName,
			acceptHeader:         jsonType,
		},
	}

	return c, nil
}
