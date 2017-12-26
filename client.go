package cbapiclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/newrelic/go-agent"
	"github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const (
	requestIDHeader      = "Request-ID"
	callingServiceHeader = "Calling-Service"
	jsonType             = "application/json"
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

// Client encapsulates the http Request functionality
type Client struct {
	// Prototype will be saturated when the Request succeeds.
	Prototype interface{}

	// ErrorPrototype will be saturated when the Request fails.
	// A Request is implicitly considered a failure if the
	// status code of the Response is not in the 2XX range.
	ErrorPrototype interface{}

	// Endpoint is the destination for the http Request
	Endpoint *url.URL

	// Request is the underlying fasthttp Request
	Request *fasthttp.Request

	// Response is the underlying fasthttp Response
	Response *fasthttp.Response

	// NRTxnName is the explicit New Relic transaction name
	NRTxnName string

	// CustomPrototypes is a map of interfaces that
	// will be saturated when specific response codes
	// are returned from the endpoint
	CustomPrototypes map[int]interface{}

	// Internal circuit breaker
	cb CircuitBreakerPrototype

	// Stack depth for new relic segment.  This tracks
	// the number of levels the client is in relation
	// to the caller
	nrStackDepth int

	// if the http response code is < 200 or > 299, this flag
	// gets set true
	responseIsError bool

	// do not automatically recycle the fasthttp request and response
	keepFastHttpArtifacts bool
}

var (
	pkgServiceName           string
	pkgUserAgent             string
	pkgNRTxnProviderFunc     func(ctx context.Context) (newrelic.Transaction, bool)
	pkgCtxLoggerProviderFunc func(ctx context.Context) (*logrus.Entry, bool)
	pkgRequestIDProviderFunc func(cxt context.Context) (string, bool)
	pkgOnce                  sync.Once
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

// NewClient will initialize and return a new client with a
// fasthttp request and endpoint.  The client's content type
// defaults to application/json
func NewClient(uri string) (*Client, error) {
	ep, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Request:  fasthttp.AcquireRequest(),
		Endpoint: ep,
	}

	c.Request.SetRequestURI(ep.String())
	c.Request.Header.SetUserAgent(pkgUserAgent)
	c.Request.Header.Set(callingServiceHeader, pkgServiceName)
	c.Request.Header.SetContentType(jsonType)

	return c, nil
}

//
// Client Functions
// ========================================================
//

// applyPreflightHeaders will apply headers right before
// the request is launched
func (c *Client) applyPreflightHeaders(ctx context.Context) {
	if requestID, ok := pkgRequestIDProviderFunc(ctx); ok {
		c.Request.Header.Set(requestIDHeader, requestID)
	}
}

// startNewRelicSegment will create a new New Relic measurement
// segment for the request.  It will reuse a New Relic transaction
// provided in SetDefaults.  Otherwise it will start a new transaction.
func (c *Client) startNewRelicSegment(ctx context.Context) newrelic.Segment {

	// get new relic transaction from context
	txn, _ := pkgNRTxnProviderFunc(ctx)

	if c.NRTxnName != "" {
		return newrelic.StartSegment(txn, c.NRTxnName)
	}

	c.nrStackDepth++
	pc, _, _, _ := runtime.Caller(c.nrStackDepth)
	funcPath := strings.Split(runtime.FuncForPC(pc).Name(), "/")

	return newrelic.StartSegment(txn, strings.Split(funcPath[len(funcPath)-1], ".")[1]+c.Endpoint.RawPath)
}

// doInternal will perform the actual request.  This function
// is either called from within a circuit breaker, or directly
// from Do.
func (c *Client) doInternal(ctx context.Context, payload interface{}) (int, error) {
	// this function runs only on the first request
	// subsequent calls are a no-ops
	ensurePackageVariables()

	logger, canLog := pkgCtxLoggerProviderFunc(ctx)
	if canLog {
		logger = logger.WithFields(logrus.Fields{
			"payload": map[string]interface{}{
				"url": c.Endpoint.String(),
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
	c.applyPreflightHeaders(ctx)

	// process the payload if it exists
	if payload != nil {

		// if it's a json Request, marshal the payload
		if string(c.Request.Header.ContentType()) == jsonType {
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

		c.Request.SetBody(payloadBytes)
		c.Request.Header.SetContentLength(len(payloadBytes))
	}

	if canLog {
		logger.Debug("launching Request")
	}

	// create a response container
	resp := fasthttp.AcquireResponse()
	if err = fasthttp.Do(c.Request, resp); err != nil {
		if canLog {
			logger.WithField("error_message", err.Error()).Error("Request failed")
		}
		return http.StatusInternalServerError, err
	}
	// request did not fail on this side, assign response
	c.Response = resp

	statusCode := c.Response.StatusCode()
	body := c.Response.Body()

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
		if c.CustomPrototypes[statusCode] != nil {
			unmarshalTo = c.CustomPrototypes[statusCode]
		} else if notSuccess {
			// request returned error code
			unmarshalTo = c.ErrorPrototype
		} else {
			// request succeeded
			unmarshalTo = c.Prototype
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
	c.nrStackDepth++
	c.Request.Header.SetMethod(method)

	if c.Endpoint == nil {
		return http.StatusInternalServerError, errors.New("endpoint for Request not set")
	}

	// recycle artifacts unless explicitly retained
	if !c.keepFastHttpArtifacts {
		defer c.Recycle()
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

// RawResponse is a shortcut to access the raw bytes returned
// in the http response
func (c *Client) RawResponse() []byte {
	if c.Response == nil {
		return []byte{}
	}

	return c.Response.Body()
}

// StatusCodeIsError is a shortcut to determine if the status code is
// considered an error
func (c *Client) StatusCodeIsError() bool {
	return c.responseIsError
}

// Recycle will allow fasthttp to recycle the request/response back to their
// appropriate pools, which reduces GC pressure and usually improves performance
func (c *Client) Recycle() {
	if c.Request != nil {
		junk := c.Request
		c.Request = nil
		fasthttp.ReleaseRequest(junk)
	}
	if c.Response != nil {
		junk := c.Response
		c.Response = nil
		fasthttp.ReleaseResponse(junk)
	}
}

//
// Fluent Functions
// These functions may be chained together
// ========================================================
//

// WillSaturate assigns the interface that will be saturated
// when the request succeeds.  It is assumed that the value passed
// into this function can be saturated via the unmarshalling of json.
// If that is not the case, you will need to process the raw bytes
// returned in the response instead
func (c *Client) WillSaturate(proto interface{}) *Client {
	c.Prototype = proto

	return c
}

// WillSaturateOnError assigns the interface that will be saturated
// when the request fails.  It is assumed that the value passed
// into this function can be saturated via the unmarshalling of json.
// If that is not the case, you will need to process the raw bytes
// returned in the response instead.  This library treats an error
// as any response with a status code not in the 2XX range.
func (c *Client) WillSaturateOnError(proto interface{}) *Client {
	c.ErrorPrototype = proto

	return c
}

// WillSaturateWithStatusCode assigns the interface that will be
// saturated when a specific response code is encountered.
// This overrides the value of WillSaturate or WillSaturateOnError
// for the same code.  For example, if a value is passed into this
// function that should saturate on a 200 response code, that will
// take precedence over anything set in WillSaturate, but will only
// return the saturated value for a 200, and no other 2XX-level code,
// unless specified here.
func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{}) *Client {
	if c.CustomPrototypes == nil {
		c.CustomPrototypes = make(map[int]interface{}, 1)
	}

	c.CustomPrototypes[statusCode] = proto

	return c
}

// SetCircuitBreaker sets the optional circuit breaker interface that
// wraps the http request.
func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype) *Client {
	c.cb = cb

	return c
}

// SetNRTxnName will set the New Relic transaction name
func (c *Client) SetNRTxnName(name string) *Client {
	c.NRTxnName = name

	return c
}

// KeepArtifacts will keep the FastHTTP request and response from being
// recycled into the pool.  This will allow direct access to the FastHTTP
// objects.  This function MUST be called if you want to access the
// response raw bytes returned by RawResponse()
func (c *Client) KeepArtifacts() *Client {
	c.keepFastHttpArtifacts = true

	return c
}

// SetContentType will set the request content type.  By default, all
// requests are of type application/json.  If you wish to use a
// different type, here is where you override it.  Also note that if
// you do provide a content type, your payload for POST, PUT, or PATCH
// must be a byte slice or it must be convertible to a byte slice
func (c *Client) SetContentType(ct string) *Client {
	c.Request.Header.SetContentType(ct)

	return c
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

	return c.Do(ctx, http.MethodPost, payload)
}

// Put performs an HTTP PUT request with the specified payload
func (c *Client) Put(ctx context.Context, payload interface{}) (int, error) {
	c.nrStackDepth++

	return c.Do(ctx, http.MethodPut, payload)
}

// Patch performs an HTTP PATCH request with the specified payload
func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error) {
	c.nrStackDepth++

	return c.Do(ctx, http.MethodPatch, payload)
}

// Delete performs an HTTP DELETE request
func (c *Client) Delete(ctx context.Context) (int, error) {
	c.nrStackDepth++

	return c.Do(ctx, http.MethodDelete, nil)
}
