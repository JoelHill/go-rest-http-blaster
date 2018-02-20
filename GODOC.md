

# cbapiclient
`import "github.com/InVisionApp/cbapiclient"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>



## <a name="pkg-index">Index</a>
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
  * [func (c *Client) SetTimeoutMS(timeout time.Duration)](#Client.SetTimeoutMS)
  * [func (c *Client) StatusCodeIsError() bool](#Client.StatusCodeIsError)
  * [func (c *Client) WillSaturate(proto interface{})](#Client.WillSaturate)
  * [func (c *Client) WillSaturateOnError(proto interface{})](#Client.WillSaturateOnError)
  * [func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{})](#Client.WillSaturateWithStatusCode)
* [type Defaults](#Defaults)
* [type IClient](#IClient)


#### <a name="pkg-files">Package files</a>
[client.go](https://github.com/InVisionApp/cbapiclient/blob/master/client.go) 





## <a name="SetDefaults">func</a> [SetDefaults](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=8661:8697#L267)
``` go
func SetDefaults(defaults *Defaults)
```
SetDefaults will apply package-level default values to
be used on all requests




## <a name="CircuitBreakerPrototype">type</a> [CircuitBreakerPrototype](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=1337:1438#L43)
``` go
type CircuitBreakerPrototype interface {
    Execute(func() (interface{}, error)) (interface{}, error)
}
```
CircuitBreakerPrototype defines the circuit breaker Execute function signature










## <a name="Client">type</a> [Client](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=3983:5309#L114)
``` go
type Client struct {
    // contains filtered or unexported fields
}
```
Client encapsulates the http Request functionality







### <a name="NewClient">func</a> [NewClient](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=10000:10043#L308)
``` go
func NewClient(uri string) (*Client, error)
```
NewClient will initialize and return a new client with a
request and endpoint.  The client's content type defaults
to application/json





### <a name="Client.Delete">func</a> (\*Client) [Delete](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21557:21614#L708)
``` go
func (c *Client) Delete(ctx context.Context) (int, error)
```
Delete performs an HTTP DELETE request




### <a name="Client.Do">func</a> (\*Client) [Do](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=16297:16386#L532)
``` go
func (c *Client) Do(ctx context.Context, method string, payload interface{}) (int, error)
```
Do will prepare the request and either run it directly
or from within a circuit breaker




### <a name="Client.Duration">func</a> (\*Client) [Duration](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20489:20530#L667)
``` go
func (c *Client) Duration() time.Duration
```
Duration will return the elapsed time of the request in an
int64 nanosecond count




### <a name="Client.Get">func</a> (\*Client) [Get](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20683:20737#L677)
``` go
func (c *Client) Get(ctx context.Context) (int, error)
```
Get performs an HTTP GET request




### <a name="Client.KeepRawResponse">func</a> (\*Client) [KeepRawResponse](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=17100:17134#L564)
``` go
func (c *Client) KeepRawResponse()
```
KeepRawResponse will cause the raw bytes from the http response
to be retained




### <a name="Client.Patch">func</a> (\*Client) [Patch](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21339:21416#L700)
``` go
func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error)
```
Patch performs an HTTP PATCH request with the specified payload




### <a name="Client.Post">func</a> (\*Client) [Post](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20866:20942#L684)
``` go
func (c *Client) Post(ctx context.Context, payload interface{}) (int, error)
```
Post performs an HTTP POST request with the specified payload




### <a name="Client.Put">func</a> (\*Client) [Put](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=21102:21177#L692)
``` go
func (c *Client) Put(ctx context.Context, payload interface{}) (int, error)
```
Put performs an HTTP PUT request with the specified payload




### <a name="Client.RawResponse">func</a> (\*Client) [RawResponse](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=17252:17289#L570)
``` go
func (c *Client) RawResponse() []byte
```
RawResponse is a shortcut to access the raw bytes returned
in the http response




### <a name="Client.SetCircuitBreaker">func</a> (\*Client) [SetCircuitBreaker](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=19473:19535#L631)
``` go
func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype)
```
SetCircuitBreaker sets the optional circuit breaker interface that
wraps the http request.




### <a name="Client.SetContentType">func</a> (\*Client) [SetContentType](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20016:20058#L645)
``` go
func (c *Client) SetContentType(ct string)
```
SetContentType will set the request content type.  By default, all
requests are of type application/json.  If you wish to use a
different type, here is where you override it.  Also note that if
you do provide a content type, your payload for POST, PUT, or PATCH
must be a byte slice or it must be convertible to a byte slice




### <a name="Client.SetHeader">func</a> (\*Client) [SetHeader](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=20249:20301#L656)
``` go
func (c *Client) SetHeader(key string, value string)
```
SetHeader allows for custom http headers




### <a name="Client.SetNRTxnName">func</a> (\*Client) [SetNRTxnName](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=19608:19650#L636)
``` go
func (c *Client) SetNRTxnName(name string)
```
SetNRTxnName will set the New Relic transaction name




### <a name="Client.SetTimeoutMS">func</a> (\*Client) [SetTimeoutMS](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=17509:17561#L580)
``` go
func (c *Client) SetTimeoutMS(timeout time.Duration)
```
SetTimeoutMS sets the maximum number of milliseconds allowed for
a request to complete.  The default request timeout is 8 seconds (8000 ms)




### <a name="Client.StatusCodeIsError">func</a> (\*Client) [StatusCodeIsError](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=17743:17784#L590)
``` go
func (c *Client) StatusCodeIsError() bool
```
StatusCodeIsError is a shortcut to determine if the status code is
considered an error




### <a name="Client.WillSaturate">func</a> (\*Client) [WillSaturate](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=18117:18165#L599)
``` go
func (c *Client) WillSaturate(proto interface{})
```
WillSaturate assigns the interface that will be saturated
when the request succeeds.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead




### <a name="Client.WillSaturateOnError">func</a> (\*Client) [WillSaturateOnError](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=18588:18643#L609)
``` go
func (c *Client) WillSaturateOnError(proto interface{})
```
WillSaturateOnError assigns the interface that will be saturated
when the request fails.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead.  This library treats an error
as any response with a status code not in the 2XX range.




### <a name="Client.WillSaturateWithStatusCode">func</a> (\*Client) [WillSaturateWithStatusCode](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=19164:19242#L621)
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




## <a name="Defaults">type</a> [Defaults](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=1500:3157#L48)
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
}
```
Defaults is a container for setting package level values










## <a name="IClient">type</a> [IClient](https://github.com/InVisionApp/cbapiclient/blob/master/client.go?s=3159:3927#L92)
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













- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
