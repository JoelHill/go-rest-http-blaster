

# cbapiclient
`import "github.com/InVisionApp/cbapiclient"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>



## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [func SetDefaults(defaults *Defaults)](#SetDefaults)
* [type CircuitBreakerPrototype](#CircuitBreakerPrototype)
* [type Client](#Client)
  * [func NewClient(uri string) (*Client, error)](#NewClient)
  * [func (c *Client) Delete(ctx context.Context) (int, error)](#Client.Delete)
  * [func (c *Client) Do(ctx context.Context, method string, payload interface{}) (int, error)](#Client.Do)
  * [func (c *Client) Duration() time.Duration](#Client.Duration)
  * [func (c *Client) Get(ctx context.Context) (int, error)](#Client.Get)
  * [func (c *Client) KeepRawResponse()](#Client.KeepRawResponse)
  * [func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error)](#Client.Patch)
  * [func (c *Client) Post(ctx context.Context, payload interface{}) (int, error)](#Client.Post)
  * [func (c *Client) Put(ctx context.Context, payload interface{}) (int, error)](#Client.Put)
  * [func (c *Client) RawResponse() []byte](#Client.RawResponse)
  * [func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype)](#Client.SetCircuitBreaker)
  * [func (c *Client) SetContentType(ct string)](#Client.SetContentType)
  * [func (c *Client) SetHeader(key string, value string)](#Client.SetHeader)
  * [func (c *Client) SetNRTxnName(name string)](#Client.SetNRTxnName)
  * [func (c *Client) SetStatsdClientWithTags(sd StatsdClientPrototype, tags []string)](#Client.SetStatsdClientWithTags)
  * [func (c *Client) SetTimeoutMS(timeout time.Duration)](#Client.SetTimeoutMS)
  * [func (c *Client) StatusCodeIsError() bool](#Client.StatusCodeIsError)
  * [func (c *Client) WillSaturate(proto interface{})](#Client.WillSaturate)
  * [func (c *Client) WillSaturateOnError(proto interface{})](#Client.WillSaturateOnError)
  * [func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{})](#Client.WillSaturateWithStatusCode)
* [type Defaults](#Defaults)
* [type IClient](#IClient)
* [type StatsdClientPrototype](#StatsdClientPrototype)


#### <a name="pkg-files">Package files</a>
[client.go](https://github.com/InVisionApp/cbapiclient/blob/master/client.go) 


## <a name="pkg-constants">Constants</a>
``` go
const NAME = "cbapiclient"
```
NAME is the name of this library




## <a name="SetDefaults">func</a> [SetDefaults](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=9424:9460#L294)
``` go
func SetDefaults(defaults *Defaults)
```
SetDefaults will apply package-level default values to
be used on all requests




## <a name="CircuitBreakerPrototype">type</a> [CircuitBreakerPrototype](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=1401:1502#L46)
``` go
type CircuitBreakerPrototype interface {
    Execute(func() (interface{}, error)) (interface{}, error)
}
```
CircuitBreakerPrototype defines the circuit breaker Execute function signature










## <a name="Client">type</a> [Client](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=4433:5928#L127)
``` go
type Client struct {
    // contains filtered or unexported fields
}
```
Client encapsulates the http Request functionality







### <a name="NewClient">func</a> [NewClient](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=10800:10843#L336)
``` go
func NewClient(uri string) (*Client, error)
```
NewClient will initialize and return a new client with a
request and endpoint.  The client's content type defaults
to application/json





### <a name="Client.Delete">func</a> (\*Client) [Delete](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=24599:24656#L818)
``` go
func (c *Client) Delete(ctx context.Context) (int, error)
```
Delete performs an HTTP DELETE request




### <a name="Client.Do">func</a> (\*Client) [Do](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=18852:18941#L633)
``` go
func (c *Client) Do(ctx context.Context, method string, payload interface{}) (int, error)
```
Do will prepare the request and either run it directly
or from within a circuit breaker




### <a name="Client.Duration">func</a> (\*Client) [Duration](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=23531:23572#L777)
``` go
func (c *Client) Duration() time.Duration
```
Duration will return the elapsed time of the request in an
int64 nanosecond count




### <a name="Client.Get">func</a> (\*Client) [Get](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=23725:23779#L787)
``` go
func (c *Client) Get(ctx context.Context) (int, error)
```
Get performs an HTTP GET request




### <a name="Client.KeepRawResponse">func</a> (\*Client) [KeepRawResponse](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=19614:19648#L663)
``` go
func (c *Client) KeepRawResponse()
```
KeepRawResponse will cause the raw bytes from the http response
to be retained




### <a name="Client.Patch">func</a> (\*Client) [Patch](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=24381:24458#L810)
``` go
func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error)
```
Patch performs an HTTP PATCH request with the specified payload




### <a name="Client.Post">func</a> (\*Client) [Post](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=23908:23984#L794)
``` go
func (c *Client) Post(ctx context.Context, payload interface{}) (int, error)
```
Post performs an HTTP POST request with the specified payload




### <a name="Client.Put">func</a> (\*Client) [Put](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=24144:24219#L802)
``` go
func (c *Client) Put(ctx context.Context, payload interface{}) (int, error)
```
Put performs an HTTP PUT request with the specified payload




### <a name="Client.RawResponse">func</a> (\*Client) [RawResponse](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=19766:19803#L669)
``` go
func (c *Client) RawResponse() []byte
```
RawResponse is a shortcut to access the raw bytes returned
in the http response




### <a name="Client.SetCircuitBreaker">func</a> (\*Client) [SetCircuitBreaker](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21987:22049#L730)
``` go
func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype)
```
SetCircuitBreaker sets the optional circuit breaker interface that
wraps the http request.




### <a name="Client.SetContentType">func</a> (\*Client) [SetContentType](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=23058:23100#L755)
``` go
func (c *Client) SetContentType(ct string)
```
SetContentType will set the request content type.  By default, all
requests are of type application/json.  If you wish to use a
different type, here is where you override it.  Also note that if
you do provide a content type, your payload for POST, PUT, or PATCH
must be a byte slice or it must be convertible to a byte slice




### <a name="Client.SetHeader">func</a> (\*Client) [SetHeader](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=23291:23343#L766)
``` go
func (c *Client) SetHeader(key string, value string)
```
SetHeader allows for custom http headers




### <a name="Client.SetNRTxnName">func</a> (\*Client) [SetNRTxnName](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=22650:22692#L746)
``` go
func (c *Client) SetNRTxnName(name string)
```
SetNRTxnName will set the New Relic transaction name




### <a name="Client.SetStatsdClientWithTags">func</a> (\*Client) [SetStatsdClientWithTags](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=22376:22457#L740)
``` go
func (c *Client) SetStatsdClientWithTags(sd StatsdClientPrototype, tags []string)
```
SetStatsdClientWithTags will set the request's statsd client and all associated tags.
Note that cbapiclient add its own tag in the form of


	cbapiclient:{VERB}-{HOST}-{PATH}

It will also add the following tags for responses from http requests:


	status_code:{STATUS_CODE}
	duration:{DURATION}




### <a name="Client.SetTimeoutMS">func</a> (\*Client) [SetTimeoutMS](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20023:20075#L679)
``` go
func (c *Client) SetTimeoutMS(timeout time.Duration)
```
SetTimeoutMS sets the maximum number of milliseconds allowed for
a request to complete.  The default request timeout is 8 seconds (8000 ms)




### <a name="Client.StatusCodeIsError">func</a> (\*Client) [StatusCodeIsError](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20257:20298#L689)
``` go
func (c *Client) StatusCodeIsError() bool
```
StatusCodeIsError is a shortcut to determine if the status code is
considered an error




### <a name="Client.WillSaturate">func</a> (\*Client) [WillSaturate](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20631:20679#L698)
``` go
func (c *Client) WillSaturate(proto interface{})
```
WillSaturate assigns the interface that will be saturated
when the request succeeds.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead




### <a name="Client.WillSaturateOnError">func</a> (\*Client) [WillSaturateOnError](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21102:21157#L708)
``` go
func (c *Client) WillSaturateOnError(proto interface{})
```
WillSaturateOnError assigns the interface that will be saturated
when the request fails.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead.  This library treats an error
as any response with a status code not in the 2XX range.




### <a name="Client.WillSaturateWithStatusCode">func</a> (\*Client) [WillSaturateWithStatusCode](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21678:21756#L720)
``` go
func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{})
```
WillSaturateWithStatusCode assigns the interface that will be
saturated when a specific response code is encountered.
This overrides the value of WillSaturate or WillSaturateOnError
for the same code.  For example, if a value is passed into this
function that should saturate on a 200 response code, that will
take precedence over anything set in WillSaturate, but will only
return the saturated value for a 200, and no other 2XX-level code,
unless specified here.




## <a name="Defaults">type</a> [Defaults](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=1819:3541#L57)
``` go
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

    // StatsdRate is the statsd reporting rate
    StatsdRate float64
}
```
Defaults is a container for setting package level values










## <a name="IClient">type</a> [IClient](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=3543:4377#L104)
``` go
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
    SetStatsdClientWithTags(sd StatsdClientPrototype, tags []string)
    SetContentType(ct string)
    SetHeader(key string, value string)
    SetNRTxnName(name string)
    SetTimeoutMS(timeout time.Duration)
    StatusCodeIsError() bool
    WillSaturate(proto interface{})
    WillSaturateOnError(proto interface{})
    WillSaturateWithStatusCode(statusCode int, proto interface{})
}
```









## <a name="StatsdClientPrototype">type</a> [StatsdClientPrototype](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=1586:1757#L51)
``` go
type StatsdClientPrototype interface {
    Incr(name string, tags []string, rate float64) error
    Timing(name string, value time.Duration, tags []string, rate float64) error
}
```
StatsdClientPrototype defines the statsd client functions used in this library














- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
