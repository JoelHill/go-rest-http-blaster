

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
  * [func (c *Client) Get(ctx context.Context) (int, error)](#Client.Get)
  * [func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error)](#Client.Patch)
  * [func (c *Client) Post(ctx context.Context, payload interface{}) (int, error)](#Client.Post)
  * [func (c *Client) Put(ctx context.Context, payload interface{}) (int, error)](#Client.Put)
  * [func (c *Client) RawResponse() []byte](#Client.RawResponse)
  * [func (c *Client) Recycle()](#Client.Recycle)
  * [func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype) *Client](#Client.SetCircuitBreaker)
  * [func (c *Client) SetContentType(ct string) *Client](#Client.SetContentType)
  * [func (c *Client) StatusCodeIsError() bool](#Client.StatusCodeIsError)
  * [func (c *Client) WillSaturate(proto interface{}) *Client](#Client.WillSaturate)
  * [func (c *Client) WillSaturateOnError(proto interface{}) *Client](#Client.WillSaturateOnError)
  * [func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{}) *Client](#Client.WillSaturateWithStatusCode)
* [type Defaults](#Defaults)


#### <a name="pkg-files">Package files</a>
[client.go](/src/github.com/InVisionApp/cbapiclient/client.go) 





## <a name="SetDefaults">func</a> [SetDefaults](/src/target/client.go?s=4862:4898#L161)
``` go
func SetDefaults(defaults *Defaults)
```
SetDefaults will apply package-level default values to
be used on all requests




## <a name="CircuitBreakerPrototype">type</a> [CircuitBreakerPrototype](/src/target/client.go?s=476:577#L28)
``` go
type CircuitBreakerPrototype interface {
    Execute(func() (interface{}, error)) (interface{}, error)
}
```
CircuitBreakerPrototype defines the circuit breaker Execute function signature










## <a name="Client">type</a> [Client](/src/target/client.go?s=1634:2640#L57)
``` go
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

    // CustomPrototypes is a map of interfaces that
    // will be saturated when specific response codes
    // are returned from the endpoint
    CustomPrototypes map[int]interface{}
    // contains filtered or unexported fields
}
```
Client encapsulates the http Request functionality







### <a name="NewClient">func</a> [NewClient](/src/target/client.go?s=5283:5326#L171)
``` go
func NewClient(uri string) (*Client, error)
```
NewClient will initialize and return a new client with a
fasthttp request and endpoint.  The client's content type
defaults to application/json





### <a name="Client.Delete">func</a> (\*Client) [Delete](/src/target/client.go?s=14269:14326#L485)
``` go
func (c *Client) Delete(ctx context.Context) (int, error)
```
Delete performs an HTTP DELETE request




### <a name="Client.Do">func</a> (\*Client) [Do](/src/target/client.go?s=9755:9844#L334)
``` go
func (c *Client) Do(ctx context.Context, method string, payload interface{}) (int, error)
```
Do will prepare the request and either run it directly
or from within a circuit breaker




### <a name="Client.Get">func</a> (\*Client) [Get](/src/target/client.go?s=13479:13533#L457)
``` go
func (c *Client) Get(ctx context.Context) (int, error)
```
Get performs an HTTP GET request




### <a name="Client.Patch">func</a> (\*Client) [Patch](/src/target/client.go?s=14080:14157#L478)
``` go
func (c *Client) Patch(ctx context.Context, payload interface{}) (int, error)
```
Patch performs an HTTP PATCH request with the specified payload




### <a name="Client.Post">func</a> (\*Client) [Post](/src/target/client.go?s=13662:13738#L464)
``` go
func (c *Client) Post(ctx context.Context, payload interface{}) (int, error)
```
Post performs an HTTP POST request with the specified payload




### <a name="Client.Put">func</a> (\*Client) [Put](/src/target/client.go?s=13870:13945#L471)
``` go
func (c *Client) Put(ctx context.Context, payload interface{}) (int, error)
```
Put performs an HTTP PUT request with the specified payload




### <a name="Client.RawResponse">func</a> (\*Client) [RawResponse](/src/target/client.go?s=10298:10335#L356)
``` go
func (c *Client) RawResponse() []byte
```
RawResponse is a shortcut to access the raw bytes returned
in the http response




### <a name="Client.Recycle">func</a> (\*Client) [Recycle](/src/target/client.go?s=10737:10763#L372)
``` go
func (c *Client) Recycle()
```
Recycle will allow fasthttp to recycle the request/response back to their
appropriate pools, which reduces GC pressure and usually improves performance




### <a name="Client.SetCircuitBreaker">func</a> (\*Client) [SetCircuitBreaker](/src/target/client.go?s=12809:12879#L434)
``` go
func (c *Client) SetCircuitBreaker(cb CircuitBreakerPrototype) *Client
```
SetCircuitBreaker sets the optional circuit breaker interface that
wraps the http request.




### <a name="Client.SetContentType">func</a> (\*Client) [SetContentType](/src/target/client.go?s=13247:13297#L445)
``` go
func (c *Client) SetContentType(ct string) *Client
```
SetContentType will set the request content type.  By default, all
requests are of type application/json.  If you wish to use a
different type, here is where you override it.  Also note that if
you do provide a content type, your payload for POST, PUT, or PATCH
must be a byte slice or it must be convertible to a byte slice




### <a name="Client.StatusCodeIsError">func</a> (\*Client) [StatusCodeIsError](/src/target/client.go?s=10506:10547#L366)
``` go
func (c *Client) StatusCodeIsError() bool
```
StatusCodeIsError is a shortcut to determine if the status code is
considered an error




### <a name="Client.WillSaturate">func</a> (\*Client) [WillSaturate](/src/target/client.go?s=11396:11452#L396)
``` go
func (c *Client) WillSaturate(proto interface{}) *Client
```
WillSaturate assigns the interface that will be saturated
when the request succeeds.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead




### <a name="Client.WillSaturateOnError">func</a> (\*Client) [WillSaturateOnError](/src/target/client.go?s=11886:11949#L408)
``` go
func (c *Client) WillSaturateOnError(proto interface{}) *Client
```
WillSaturateOnError assigns the interface that will be saturated
when the request fails.  It is assumed that the value passed
into this function can be saturated via the unmarshalling of json.
If that is not the case, you will need to process the raw bytes
returned in the response instead.  This library treats an error
as any response with a status code not in the 2XX range.




### <a name="Client.WillSaturateWithStatusCode">func</a> (\*Client) [WillSaturateWithStatusCode](/src/target/client.go?s=12481:12567#L422)
``` go
func (c *Client) WillSaturateWithStatusCode(statusCode int, proto interface{}) *Client
```
WillSaturateWithStatusCode assigns the interface that will be
saturated when a specific response code is encountered.
This overrides the value of WillSaturate or WillSaturateOnError
for the same code.  For example, if a value is passed into this
function that should saturate on a 200 response code, that will
take precedence over anything set in WillSaturate, but will only
return the saturated value for a 200, and no other 2XX-level code,
unless specified here.




## <a name="Defaults">type</a> [Defaults](/src/target/client.go?s=639:1578#L33)
``` go
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
```
Defaults is a container for setting package level values














- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
