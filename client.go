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

	"github.com/Sirupsen/logrus"
	"github.com/newrelic/go-agent"
	"github.com/nu7hatch/gouuid"
)

//go:generate counterfeiter -o ./fakes/fake_circuitbreaker_prototype.go . CircuitBreakerPrototype
//go:generate counterfeiter -o ./fakes/fake_iclient.go . IClient

const (
	requestIDHeader      = "Request-ID"
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
}

type IClient interface {
	Delete(ctx context.Context) (int, error)
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

var (
	pkgServiceName           string
	pkgUserAgent             string
	pkgNRTxnProviderFunc     func(ctx context.Context) (newrelic.Transaction, bool)
	pkgCtxLoggerProviderFunc func(ctx context.Context) (*logrus.Entry, bool)
	pkgRequestIDProviderFunc func(cxt context.Context) (string, bool)
	pkgOnce                  sync.Once

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

		// user agent is service name + namespace + tenancy
		pkgUserAgent = fmt.Sprintf(
			"%s-%s-%s",
			pkgServiceName,
			os.Getenv("NAMESPACE"),
			os.Getenv("TENANCY"))

		// make sure new relic transaction provider exists
		if pkgNRTxnProviderFunc == nil {
			logrus.Warn("Client: No NewRelicTransactionProviderFunc default set.")
			pkgNRTxnProviderFunc = func(ctx context.Context) (newrelic.Transaction, bool) {
				// the newrelic StartSegment function will start a new transaction
				return nil, false
			}
		}

		// make sure the context logger provider exists
		if pkgCtxLoggerProviderFunc == nil {
			logrus.Warn("APIRequest: No ContextLoggerProviderFunc default set.  A new logger will be used")
			pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
				return logrus.WithFields(logrus.Fields{}), true
			}
		}

		// make sure the Request id provider exists
		if pkgRequestIDProviderFunc == nil {
			logrus.Warn("APIRequest: No RequestIDProviderFunc default set.  Generating new Request id")
			pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
				// error can be safely ignored
				reqUUID, _ := uuid.NewV4()
				return reqUUID.String(), true
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
// fasthttp request and endpoint.  The client's content type
// defaults to application/json
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
	} else {
		// some apis will fail if the request id is not set,
		// so we will stub one in here
		reqUUID, _ := uuid.NewV4()
		requestID = reqUUID.String()
		logrus.WithFields(logrus.Fields{
			"error_message": "REQUEST ID ERROR: the provided request ID provider is not returning a request id",
		}).Errorf("REQUEST ID: %s", requestID)
		c.headers[requestIDHeader] = requestID
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
	if canLog {
		logger = logger.WithFields(logrus.Fields{
			"func_name": "cbapiclient.doInternal",
			"payload": map[string]interface{}{
				"url": c.endpoint.String(),
			},
		})
	}

	// start the new relic capture
	segment := c.startNewRelicSegment(ctx)
	defer segment.End()

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
					logger.WithField("error_message", err.Error()).Error("Request failed")
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
				if canLog {
					logger.WithField("error_message", err.Error()).Error("Request failed")
				}
				return http.StatusInternalServerError, errors.New("the payload cannot be converted to a byte slice")
			}
		}

		c.headers[contentLengthHeader] = fmt.Sprintf("%d", len(payloadBytes))
	}

	if canLog {
		logger.Debug("launching Request")
	}

	request, err := http.NewRequest(c.method, c.endpoint.String(), bytes.NewReader(payloadBytes))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	for k, v := range c.headers {
		request.Header.Set(k, v)
	}

	response, err := c.client.Do(request)
	// close request body immediately
	if reqCloseErr := request.Body.Close(); reqCloseErr != nil {
		logger.Errorf("error closing request io reader: %v", reqCloseErr)
	}
	// request error
	if err != nil {
		return http.StatusInternalServerError, err
	}
	// defer response body reader close
	defer func(resp *http.Response, logger *logrus.Entry) {
		if err := resp.Body.Close(); err != nil {
			logger.Errorf("error ")
		}
	}(response, logger)

	// get status code
	statusCode := response.StatusCode

	// get response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return http.StatusInternalServerError, err
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
		notSuccess := statusCode < 200 || statusCode > 299
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
					logger.WithField("error_message", err.Error()).Error("Request failed")
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
		return http.StatusInternalServerError, errors.New("endpoint for Request not set")
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
