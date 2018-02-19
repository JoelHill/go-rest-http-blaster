package cbapiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/newrelic/go-agent"
	"github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
)

//go:generate counterfeiter -o ./fakes/fake_circuitbreaker_prototype.go . CircuitBreakerPrototype
//go:generate counterfeiter -o ./fakes/fake_iclient.go . IClient

const (
	requestIDHeader      = "Request-ID"
	requestSourceHeader  = "Request-Source"
	callingServiceHeader = "Calling-Service"
	jsonType             = "application/json"
	contentTypeHeader    = "Content-Type"
	userAgentHeader      = "User-Agent"
	contentLengthHeader  = "Content-Length"
	acceptHeader         = "Accept"
	requestTimeout       = 8 * time.Second  // the max amount of time for the entire request before failing
	sockTimeout          = 2 * time.Second  // the max amount of time attempting to make the tcp connection
	tlsTimeout           = 2 * time.Second  // the max amount of time establishing TLS handshake
	idleTimeout          = 10 * time.Second // the amount of time to keep idle connections available before closing them
	maxIdleConnsPerHost  = 100              // the maximum number of idle connections to keep around for reuse
)

// CircuitBreakerPrototype defines the circuit breaker Execute function signature
type CircuitBreakerPrototype interface {
	Execute(func() (interface{}, error)) (interface{}, error)
}

// Defaults is a container for setting package level values
type Defaults struct {
	// ServiceName is the name of the calling service
	ServiceName string

	// DontUseNewRelic will disable the New Relic transaction
	// segment if it's not available or wanted.  This is useful
	// for testing purposes and/or prototyping.  We should be
	// using New Relic transaction wrappers if they are available.
	DontUseNewRelic bool

	// NewRelicTransactionProviderFunc is a function that
	// provides the New Relic transaction to be used in the
	// HTTP Request.  If this function is not set, the client
	// will create a new New Relic transaction
	NewRelicTransactionProviderFunc func(ctx context.Context) (newrelic.Transaction, bool)

	// ContextLoggerProviderFunc is a function that provides
	// a logger from the current context.  If this function
	// is not set, the client will create a new logger for
	// the Request.
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
}

type IClient interface {
	Delete(ctx context.Context) (int, error)
	Duration() time.Duration
	Do(ctx context.Context, method string, payload interface{}) (int, error)
	Get(ctx context.Context) (int, error)
	KeepRawResponse()
	Post(ctx context.Context, payload interface{}) (int, error)
	Put(ctx context.Context, payload interface{}) (int, error)
	Patch(ctx context.Context, payload interface{}) (int, error)
	RawResponse() []byte
	SetCircuitBreaker(cb CircuitBreakerPrototype)
	SetContentType(ct string)
	SetHeader(key string, value string)
	SetNRTxnName(name string)
	StatusCodeIsError() bool
	WillSaturate(proto interface{})
	WillSaturateOnError(proto interface{})
	WillSaturateWithStatusCode(statusCode int, proto interface{})
}

// Client encapsulates the http Request functionality
type Client struct {
	// prototype will be saturated when the Request succeeds.
	prototype interface{}

	// errorPrototype will be saturated when the Request fails.
	// A Request is implicitly considered a failure if the
	// status code of the Response is not in the 2XX range.
	errorPrototype interface{}

	// endpoint is the destination for the http Request
	endpoint *url.URL

	// nrTxnName is the explicit New Relic transaction name
	nrTxnName string

	// customPrototypes is a map of interfaces that
	// will be saturated when specific response codes
	// are returned from the endpoint
	customPrototypes map[int]interface{}

	// duration is the length of time the request took to run.
	// Obviously this only has value after the request is run.
	duration time.Duration

	// Internal circuit breaker
	cb CircuitBreakerPrototype

	// Stack depth for new relic segment.  This tracks
	// the number of levels the client is in relation
	// to the caller
	nrStackDepth int

	// if the http response code is < 200 or > 299, this flag
	// gets set true
	responseIsError bool

	// internal http client
	client *http.Client

	// internal headers
	headers map[string]string

	// request method
	method string

	// flag to copy raw response bytes from http response
	keepRawResponse bool

	// raw response bytes
	rawresponse []byte
}

// req014HeaderCheck will check for the presence of required outgoing
// headers per the InVision REQ014 documentation
//
// @see https://invision-engineering.herokuapp.com/requirements/REQ014/index.html
type req014HeaderCheck struct {
	requestID      bool
	requestSource  bool
	callingService bool
}

// check to see if REQ014 flags are closed
func (c req014HeaderCheck) ok() bool {
	return c.requestID && c.requestSource && c.callingService
}

var (
	pkgServiceName               string
	pkgUserAgent                 string
	pkgNRTxnProviderFunc         func(ctx context.Context) (newrelic.Transaction, bool)
	pkgCtxLoggerProviderFunc     func(ctx context.Context) (*logrus.Entry, bool)
	pkgRequestIDProviderFunc     func(cxt context.Context) (string, bool)
	pkgRequestSourceProviderFunc func(cxt context.Context) (string, bool)
	pkgOnce                      sync.Once
	pkgDontUseNewRelic           bool
	pkgStrictREQ014              bool

	envHttpMocking = "MOCKING_HTTP"
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

		if !pkgDontUseNewRelic {
			// make sure new relic transaction provider exists
			if pkgNRTxnProviderFunc == nil {
				logrus.Warn("cbapiclient: No NewRelicTransactionProviderFunc default set.")
				pkgNRTxnProviderFunc = func(ctx context.Context) (newrelic.Transaction, bool) {
					// the newrelic StartSegment function will start a new transaction
					return nil, false
				}
			}
		}

		// make sure the context logger provider exists
		if pkgCtxLoggerProviderFunc == nil {
			logrus.Warn("cbapiclient: No ContextLoggerProviderFunc default set.  A new logger will be " +
				"used for each request")
			pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
				return logrus.NewEntry(logrus.New()), true
			}
		}

		// make sure the Request id provider exists
		if pkgRequestIDProviderFunc == nil {
			logrus.Warn("cbapiclient: No RequestIDProviderFunc default set.  A new id will be generated " +
				"for each request")
			pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
				// error can be safely ignored
				reqUUID, _ := uuid.NewV4()
				return reqUUID.String(), true
			}
		}

		// make sure the request source provider exists
		if pkgRequestSourceProviderFunc == nil {
			logrus.Warn("cbapiclient: No RequestSourceProviderFunc default set.  The Request-Source header " +
				"will be absent in each request unless set manually")
			pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
				return "", false
			}
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
	pkgDontUseNewRelic = defaults.DontUseNewRelic
	pkgRequestSourceProviderFunc = defaults.RequestSourceProviderFunc
	pkgUserAgent = defaults.UserAgent
	pkgStrictREQ014 = defaults.StrictREQ014
}

// this creates a http client with sensible defaults
func newHttpClient() *http.Client {
	// all http mocking libraries can override the default http client,
	// but many cannot override clients that have been tuned with custom
	// transports.  If this env var is set, we return the standard
	// http client.
	if os.Getenv(envHttpMocking) != "" {
		return http.DefaultClient
	}

	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   sockTimeout,
				DualStack: true,
			}).DialContext,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			IdleConnTimeout:     idleTimeout,
			TLSHandshakeTimeout: tlsTimeout,
		},
	}
	client.Timeout = requestTimeout

	return client
}

// NewClient will initialize and return a new client with a
// request and endpoint.  The client's content type defaults
// to application/json
func NewClient(uri string) (*Client, error) {

	ensurePackageVariables()

	ep, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	c := &Client{
		endpoint: ep,
		method:   http.MethodGet,
		client:   newHttpClient(),
		headers: map[string]string{
			userAgentHeader:      pkgUserAgent,
			contentTypeHeader:    jsonType,
			callingServiceHeader: pkgServiceName,
			acceptHeader:         jsonType,
		},
	}

	return c, nil
}

//
// Client Functions
// ========================================================
//

// applyContextDependentHeaders will apply headers right before
// the request is launched
func (c *Client) applyContextDependentHeaders(ctx context.Context) {
	if requestID, ok := pkgRequestIDProviderFunc(ctx); ok {
		c.headers[requestIDHeader] = requestID
	}

	if requestSource, ok := pkgRequestSourceProviderFunc(ctx); ok {
		c.headers[requestSourceHeader] = requestSource
	}
}

// startNewRelicSegment will create a new New Relic measurement
// segment for the request.  It will reuse a New Relic transaction
// provided in SetDefaults.  Otherwise it will start a new transaction.
func (c *Client) startNewRelicSegment(ctx context.Context) newrelic.Segment {

	// get new relic transaction from context
	txn, _ := pkgNRTxnProviderFunc(ctx)

	if c.nrTxnName != "" {
		return newrelic.StartSegment(txn, c.nrTxnName)
	}

	c.nrStackDepth++
	pc, _, _, _ := runtime.Caller(c.nrStackDepth)
	funcPath := strings.Split(runtime.FuncForPC(pc).Name(), "/")

	return newrelic.StartSegment(txn, strings.Split(funcPath[len(funcPath)-1], ".")[1]+c.endpoint.RawPath)
}

// doInternal will perform the actual request.  This function
// is either called from within a circuit breaker, or directly
// from Do.
func (c *Client) doInternal(ctx context.Context, payload interface{}) (int, error) {

	logger, canLog := pkgCtxLoggerProviderFunc(ctx)

	// start the new relic capture
	if !pkgDontUseNewRelic {
		segment := c.startNewRelicSegment(ctx)
		defer segment.End()
	}

	var (
		err          error
		payloadBytes []byte
	)

	// set default headers
	c.applyContextDependentHeaders(ctx)

	// process the payload if it exists
	if payload != nil {
		// if it's a json Request, marshal the payload
		if c.headers[contentTypeHeader] == jsonType {
			payloadBytes, err = json.Marshal(payload)
			if err != nil {
				if canLog {
					logger.WithField("error_message", err.Error()).Error("cbapiclient: request failed")
				}
				return http.StatusInternalServerError, err
			}
		} else {
			// caller has supplied content-type.  it must be convertible to byte slice
			switch payload.(type) {
			case []byte:
				payloadBytes = payload.([]byte)
			case string:
				payloadBytes = []byte(payload.(string))
			default:
				err := errors.New("the payload cannot be converted to a byte slice")
				if canLog {
					logger.WithField("error_message", err.Error()).Error("cbapiclient: request failed")
				}
				return http.StatusInternalServerError, err
			}
		}

		c.headers[contentLengthHeader] = fmt.Sprintf("%d", len(payloadBytes))
	}

	if canLog {
		logger.Debugf("cbapiclient: launching %s request to %s", c.method, c.endpoint.String())
	}

	request, err := http.NewRequest(c.method, c.endpoint.String(), bytes.NewReader(payloadBytes))
	if err != nil {
		if canLog {
			logger.WithField("error_message", err.Error()).Error("cbapiclient: create request failed")
		}
		return http.StatusInternalServerError, err
	}

	check := req014HeaderCheck{}
	for k, v := range c.headers {
		request.Header.Set(k, v)
		switch k {
		case requestIDHeader:
			check.requestID = true
		case requestSourceHeader:
			check.requestSource = true
		case callingServiceHeader:
			check.callingService = true
		}
	}

	// if we are strictly enforcing request tracing
	if pkgStrictREQ014 && !check.ok() {
		err := errors.New("request tracing header requirements check failed")
		if canLog {
			logger.WithField("error_message", err.Error()).Error("cbapiclient: illegal request")
		}
		return http.StatusInternalServerError, err
	}

	response, responseErr := c.client.Do(request)
	// close request body immediately
	if reqCloseErr := request.Body.Close(); reqCloseErr != nil {
		if canLog {
			logger.WithField("error_message", reqCloseErr.Error()).Error("cbapiclient: request body close error")
		}
	}
	// request error
	if responseErr != nil {
		if canLog {
			logger.WithField("error_message", responseErr.Error()).Error("cbapiclient: request failed")
		}
		return http.StatusInternalServerError, responseErr
	}
	// defer response body reader close
	defer func(resp *http.Response, logger *logrus.Entry) {
		if closeErr := resp.Body.Close(); closeErr != nil {
			if canLog {
				logger.WithField("error_message", closeErr.Error()).Error("cbiclient: error closing response body")
			}
		}
	}(response, logger)

	// get status code
	statusCode := response.StatusCode

	// get response body
	body, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		if canLog {
			logger.WithField("error_message", readErr.Error()).Error("cbapiclient: error reading response raw bytes")
		}
		return http.StatusInternalServerError, readErr
	}

	// only keep the raw response if explicitly requested
	if c.keepRawResponse {
		c.rawresponse = body
	}

	// if the response has a body, handle it
	if len(body) > 0 {

		// the thing we are about to potentially unmarshal into
		var unmarshalTo interface{}

		// check if this is an error
		notSuccess := statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
		if notSuccess {
			c.responseIsError = true
		}

		// if there is a custom response for this specific status code
		if c.customPrototypes[statusCode] != nil {
			unmarshalTo = c.customPrototypes[statusCode]
		} else if notSuccess {
			// request returned error code
			unmarshalTo = c.errorPrototype
		} else {
			// request succeeded
			unmarshalTo = c.prototype
		}

		// if there is something that can be unmarshalled into
		if unmarshalTo != nil {
			if err = json.Unmarshal(body, unmarshalTo); err != nil {
				if canLog {
					logger.WithField("error_message", err.Error()).Error("cbapiclient: json unmarshal error")
				}
				return http.StatusInternalServerError, err
			}
		}
	}

	return statusCode, nil
}

// Do will prepare the request and either run it directly
// or from within a circuit breaker
func (c *Client) Do(ctx context.Context, method string, payload interface{}) (int, error) {

	// start the clock
	defer func(c *Client, begin time.Time) {
		c.duration = time.Now().Sub(begin)
	}(c, time.Now())

	c.nrStackDepth++

	if c.endpoint == nil {
		err := errors.New("cbapiclient: endpoint for request not set")
		logger, canLog := pkgCtxLoggerProviderFunc(ctx)
		if canLog {
			logger.WithField("error_message", err.Error()).Error("cbapiclient config error")
		}
		return http.StatusInternalServerError, err
	}

	if c.cb == nil {
		return c.doInternal(ctx, payload)
	}

	sc, err := c.cb.Execute(func() (interface{}, error) {
		c.nrStackDepth++
		return c.doInternal(ctx, payload)
	})

	return sc.(int), err
}

// KeepRawResponse will cause the raw bytes from the http response
// to be retained
func (c *Client) KeepRawResponse() {
	c.keepRawResponse = true
}

// RawResponse is a shortcut to access the raw bytes returned
// in the http response
func (c *Client) RawResponse() []byte {
	if c.keepRawResponse {
		return c.rawresponse
	}

	return []byte{}
}

// StatusCodeIsError is a shortcut to determine if the status code is
// considered an error
func (c *Client) StatusCodeIsError() bool {
	return c.responseIsError
}

// WillSaturate assigns the interface that will be saturated
// when the request succeeds.  It is assumed that the value passed
// into this function can be saturated via the unmarshalling of json.
// If that is not the case, you will need to process the raw bytes
// returned in the response instead
func (c *Client) WillSaturate(proto interface{}) {
	c.prototype = proto
}

// WillSaturateOnError assigns the interface that will be saturated
// when the request fails.  It is assumed that the value passed
// into this function can be saturated via the unmarshalling of json.
// If that is not the case, you will need to process the raw bytes
// returned in the response instead.  This library treats an error
// as any response with a status code not in the 2XX range.
func (c *Client) WillSaturateOnError(proto interface{}) {
	c.errorPrototype = proto
}

// WillSaturateWithStatusCode assigns the interface that will be
// saturated when a specific response code is encountered.
// This overrides the value of WillSaturate or WillSaturateOnError
// for the same code.  For example, if a value is passed into this
// function that should saturate on a 200 response code, that will
// take precedence over anything set in WillSaturate, but will only
// return the saturated value for a 200, and no other 2XX-level code,
// unless specified here.
func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{}) {
	if c.customPrototypes == nil {
		c.customPrototypes = make(map[int]interface{}, 1)
	}

	c.customPrototypes[statusCode] = proto
}

// SetCircuitBreaker sets the optional circuit breaker interface that
// wraps the http request.
func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype) {
	c.cb = cb
}

// SetNRTxnName will set the New Relic transaction name
func (c *Client) SetNRTxnName(name string) {
	c.nrTxnName = name
}

// SetContentType will set the request content type.  By default, all
// requests are of type application/json.  If you wish to use a
// different type, here is where you override it.  Also note that if
// you do provide a content type, your payload for POST, PUT, or PATCH
// must be a byte slice or it must be convertible to a byte slice
func (c *Client) SetContentType(ct string) {
	c.headers[contentTypeHeader] = ct

	if ct != jsonType {
		delete(c.headers, acceptHeader)
	} else {
		c.headers[acceptHeader] = jsonType
	}
}

// SetHeader allows for custom http headers
func (c *Client) SetHeader(key string, value string) {
	if key == contentTypeHeader {
		c.SetContentType(value)
		return
	}

	c.headers[key] = value
}

// Duration will return the elapsed time of the request in an
// int64 nanosecond count
func (c *Client) Duration() time.Duration {
	return c.duration
}

//
// Convenience Functions
// ========================================================
//

// Get performs an HTTP GET request
func (c *Client) Get(ctx context.Context) (int, error) {
	c.nrStackDepth++

	return c.Do(ctx, http.MethodGet, nil)
}

// Post performs an HTTP POST request with the specified payload
func (c *Client) Post(ctx context.Context, payload interface{}) (int, error) {
	c.nrStackDepth++
	c.method = http.MethodPost

	return c.Do(ctx, http.MethodPost, payload)
}

// Put performs an HTTP PUT request with the specified payload
func (c *Client) Put(ctx context.Context, payload interface{}) (int, error) {
	c.nrStackDepth++
	c.method = http.MethodPut

	return c.Do(ctx, http.MethodPut, payload)
}

// Patch performs an HTTP PATCH request with the specified payload
func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error) {
	c.nrStackDepth++
	c.method = http.MethodPatch

	return c.Do(ctx, http.MethodPatch, payload)
}

// Delete performs an HTTP DELETE request
func (c *Client) Delete(ctx context.Context) (int, error) {
	c.nrStackDepth++
	c.method = http.MethodDelete

	return c.Do(ctx, http.MethodDelete, nil)
}
