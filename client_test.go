package blaster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/InVisionApp/go-logger"
	"github.com/InVisionApp/go-logger/shims/testlog"
	"github.com/joelhill/go-rest-http-blaster/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/h2non/gock.v1"
)

var _ = Describe("Client", func() {
	type Cat struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	type Dog struct {
		Name  string `json:"name"`
		Breed string `json:"breed"`
	}
	type PetStore struct {
		Cats []Cat `json:"cats"`
		Dogs []Dog `json:"dogs"`
	}

	var (
		client         *Client
		defaults       *Defaults
		cb             *fakes.FakeCircuitBreakerPrototype
		statsd         *fakes.FakeStatsdClientPrototype
		ctx            context.Context
		endpoint       *url.URL
		logBytes       []byte
		logBuffer      *bytes.Buffer
		byteLoggerFunc func(ctx context.Context) (*logrus.Entry, bool)
		petStore       *PetStore
		petStoreBytes  []byte
		endpointStr    string
		span           opentracing.Span
	)

	BeforeEach(func() {
		ctx = context.Background()

		pkgRequireHeaders = true
		pkgUserAgent = "unit test"
		pkgServiceName = "unit test"
		endpointStr = "http://www.invisionapp.com"
		span = opentracing.StartSpan("test")

		cb = &fakes.FakeCircuitBreakerPrototype{}
		statsd = &fakes.FakeStatsdClientPrototype{}
		endpoint, _ = url.Parse(endpointStr)
		logBytes = []byte{}
		logBuffer = bytes.NewBuffer(logBytes)
		byteLoggerFunc = func(ctx context.Context) (*logrus.Entry, bool) {
			logger := logrus.New()
			logger.Out = logBuffer
			return logrus.NewEntry(logger), true
		}
		petStore = &PetStore{
			Cats: []Cat{
				{"Scruffy", "Orange"},
				{"Shadow", "Black"},
			},
			Dogs: []Dog{
				{"Lulu", "Basset"},
				{"Pippy", "Jack Russell Terrier"},
			},
		}
		petStoreBytes = []byte(`{"cats":[{"name":"Scruffy","color":"Orange"},{"name":"Shadow","color":"Black"}],"dogs":[{"name":"Lulu","breed":"Basset"},{"name":"Pippy","breed":"Jack Russell Terrier"}]}`)

		cb.ExecuteReturns(200, nil)
		statsd.IncrReturns(nil)
		statsd.TimingReturns(nil)

		// funcs
		pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
			return "unit-test-request-id", true
		}
		pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
			return "unit-test-request-source", true
		}
		pkgTracerProviderFunc = func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span) {
			return r, span
		}

		// defaults struct
		defaults = &Defaults{
			ServiceName:               "unit-test",
			StatsdRate:                1,
			RequireHeaders:            true,
			UserAgent:                 "unit-test",
			RequestIDProviderFunc:     pkgRequestIDProviderFunc,
			RequestSourceProviderFunc: pkgRequestSourceProviderFunc,
			TracerProviderFunc:        pkgTracerProviderFunc,
		}
	})

	JustBeforeEach(func() {
		SetDefaults(defaults)
		client = &Client{
			endpoint:     endpoint,
			client:       http.DefaultClient,
			statsdClient: statsd,
			statsdStat:   "fake-api-call",
			statsdTags:   []string{},
			headers: map[string]string{
				userAgentHeader:      pkgUserAgent,
				contentTypeHeader:    jsonType,
				callingServiceHeader: pkgServiceName,
				acceptHeader:         jsonType,
			},
		}
	})

	// region REQ014
	Describe("REQ014 checks", func() {
		Context("happy path", func() {
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(string(petStoreBytes))

				_, err := client.Get(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("happy path - request source header missing, not required", func() {
			JustBeforeEach(func() {
				pkgRequireHeaders = false
				pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
					return "", false
				}
			})
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(string(petStoreBytes))

				_, err := client.Get(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("happy path - calling service header missing, not required", func() {
			JustBeforeEach(func() {
				pkgRequireHeaders = false
				delete(client.headers, callingServiceHeader)
			})
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(string(petStoreBytes))

				_, err := client.Get(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("sad path - request id header missing", func() {
			JustBeforeEach(func() {
				pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
					return "", false
				}
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})

		Context("sad path - request id header blank", func() {
			JustBeforeEach(func() {
				pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
					return "", true
				}
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})

		Context("sad path - request source header missing", func() {
			JustBeforeEach(func() {
				pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
					return "", false
				}
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})

		Context("sad path - request source header blank", func() {
			JustBeforeEach(func() {
				pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
					return "", true
				}
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})

		Context("sad path - calling service header missing", func() {
			JustBeforeEach(func() {
				delete(client.headers, callingServiceHeader)
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})

		Context("sad path - calling service header blank", func() {
			JustBeforeEach(func() {
				client.headers[callingServiceHeader] = ""
			})
			It("throws an error", func() {
				_, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("request tracing header requirements check failed"))
			})
		})
	})
	// endregion

	// region processOutgoingPayload
	Describe("processOutgoingPayload", func() {
		Context("happy path", func() {
			It("processes the payload", func() {
				payloadBytes, err := client.processOutgoingPayload(petStore)
				Expect(err).To(BeNil())
				Expect(payloadBytes).To(Equal(petStoreBytes))
			})
		})
		Context("sad path - bad payload", func() {
			It("chokes on the payload", func() {
				notJson := func() bool { return true }
				_, err := client.processOutgoingPayload(notJson)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("happy path, non-json types", func() {
			It("processes the string payload", func() {
				client.SetContentType("text/plain")
				thisIsATest := "this is a test"
				payloadBytes, err := client.processOutgoingPayload(thisIsATest)
				Expect(err).To(BeNil())
				Expect(payloadBytes).To(Equal([]byte("this is a test")))
			})
			It("processes the byte payload", func() {
				client.SetContentType("text/plain")
				b := make([]byte, 16, 64)
				b[0] = '"'
				b[1] = 't'
				b[2] = 'h'
				b[3] = 'i'
				b[4] = 's'
				b[5] = ' '
				b[6] = 'i'
				b[7] = 's'
				b[8] = ' '
				b[9] = 'a'
				b[10] = ' '
				b[11] = 't'
				b[12] = 'e'
				b[13] = 's'
				b[14] = 't'
				b[15] = '"'
				payloadBytes, err := client.processOutgoingPayload(b)
				Expect(err).To(BeNil())
				Expect(payloadBytes).To(Equal(b))
			})
		})
		Context("sad path - non-json type", func() {
			It("throws an error", func() {
				client.SetContentType("text/plain")
				_, err := client.processOutgoingPayload(false)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("the payload cannot be converted to a byte slice"))
			})
		})
	})
	// endregion

	// region statsd
	Describe("statsd", func() {
		JustBeforeEach(func() {
			statsd.IncrStub = func(name string, tags []string, rate float64) error {
				logBuffer.WriteString(fmt.Sprintf("INCR:%v\n", tags))
				return nil
			}
			statsd.TimingStub = func(name string, value time.Duration, tags []string, rate float64) error {
				logBuffer.WriteString(fmt.Sprintf("TIMING:%v\n", tags))
				return nil
			}
		})
		Context("happy path", func() {
			It("reports success to statsd", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(string(petStoreBytes))

				client.Get(ctx)

				Expect(string(logBuffer.Bytes())).To(ContainSubstring("TIMING:[response-code:200 response-type:2xx http-verb:GET called-service: route:]"))
			})
		})
		Context("happy sad path", func() {
			It("reports failure to statsd", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(500).BodyString("failed")

				client.Get(ctx)

				Expect(string(logBuffer.Bytes())).To(ContainSubstring("TIMING:[response-code:500 response-type:5xx http-verb:GET called-service: route:]"))
			})
		})
	})
	// endregion

	// region processResponseData
	Describe("processResponseData", func() {
		Context("happy path - custom prototype", func() {
			It("saturates a custom response", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").Reply(http.StatusTeapot).BodyString(string(petStoreBytes)).SetHeader(contentTypeHeader, jsonType)

				ps := &PetStore{}
				client.WillSaturateWithStatusCode(http.StatusTeapot, ps)
				statusCode, err := client.Get(ctx)
				Expect(err).To(BeNil())
				Expect(statusCode).To(Equal(http.StatusTeapot))

				Expect(len(ps.Cats)).To(Equal(2))
				Expect(len(ps.Dogs)).To(Equal(2))
			})
		})
		Context("sad path - invalid json", func() {
			It("throws an error", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").Reply(200).BodyString("<NOT a json string<>><").SetHeader(contentTypeHeader, jsonType)

				ps := &PetStore{}
				client.WillSaturate(ps)
				statusCode, err := client.Get(ctx)
				Expect(err).ToNot(BeNil())
				Expect(statusCode).To(Equal(500))
			})
		})
		Context("sad path - non-json response when expecting json", func() {
			It("throws an error", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").Reply(200).BodyString("<this is html>").SetHeader(contentTypeHeader, "text/html")

				ps := &PetStore{}
				client.WillSaturate(ps)
				statusCode, err := client.Get(ctx)
				Expect(err).To(BeNil())
				Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))
				Expect(string(client.RawResponse())).To(ContainSubstring("<this is html>"))
			})
		})
	})
	// endregion

	// region doInternal
	Describe("doInternal", func() {
		Context("bad request", func() {
			It("throws an error", func() {
				client.logger = log.NewNoop()
				client.method = "bad method"
				_, err := client.doInternal(ctx, nil)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`net/http: invalid method "bad method"`))
			})
		})
		Context("bad payload", func() {
			It("throws an error", func() {
				client.logger = log.NewNoop()
				client.method = http.MethodGet
				_, err := client.doInternal(ctx, func() bool { return true })
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring(`json: unsupported type: func() bool`))
			})
		})
		Context("response error", func() {
			JustBeforeEach(func() {
				client.logger = log.NewNoop()
			})
			It("returns a general response error", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").ReplyError(errors.New("FAIL")).Status(http.StatusInternalServerError)
				client.logger = log.NewNoop()
				_, err := client.doInternal(ctx, nil)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("timeout", func() {
			JustBeforeEach(func() {
				client.logger = log.NewNoop()
			})
			It("reports a timeout error", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").ReplyError(fakes.TimeoutError{}).Status(http.StatusRequestTimeout)
				_, err := client.doInternal(ctx, nil)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("read body error", func() {
			JustBeforeEach(func() {
				client.logger = log.NewNoop()
			})
			It("chokes on the response", func() {
				defer gock.OffAll()
				resp := gock.New(endpointStr).Get("/").Reply(200)
				resp.Body(&fakes.SadIOReader{})
				_, err := client.doInternal(ctx, nil)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("raw response", func() {
			var bodyStr = `{"foo": "bar"}`
			JustBeforeEach(func() {
				client.logger = log.NewNoop()
			})
			It("keeps the raw response", func() {
				defer gock.OffAll()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(bodyStr)
				client.KeepRawResponse()
				_, err := client.doInternal(ctx, nil)
				Expect(err).To(BeNil())
				rb := client.RawResponse()
				Expect(string(rb)).To(Equal(bodyStr))
			})
		})
	})
	// endregion

	// region closeResponse
	Describe("closeResponse", func() {
		var (
			resp   *http.Response
			logger *testlog.TestLogger
		)
		BeforeEach(func() {
			resp = &http.Response{}
			logger = testlog.New()
		})
		Context("happy paths", func() {
			BeforeEach(func() {
				resp.Body = fakes.HappyIOReadCloser{}
			})
			It("closes the response without error", func() {
				closeResponse(resp, logger)
				output := logger.Bytes()
				Expect(len(output)).To(Equal(0))
			})
		})
		Context("sad paths", func() {
			BeforeEach(func() {
				resp.Body = fakes.SadIOCloser{}
			})
			It("returns an error", func() {
				closeResponse(resp, logger)
				output := logger.Bytes()
				Expect(string(output)).To(ContainSubstring("unable to close response body"))
			})
		})
	})
	// endregion

	// region Do
	Describe("Do", func() {
		BeforeEach(func() {

		})
		Context("happy paths", func() {
			Context("all good", func() {
				JustBeforeEach(func() {
					client.logger = log.NewNoop()
					client.SetCircuitBreaker(cb)
				})
				It("all passes", func() {
					statusCode, err := client.Do(ctx, "GET", nil)
					Expect(err).To(BeNil())
					Expect(statusCode).To(Equal(http.StatusOK))
				})
			})
			Context("circuit breaker", func() {
				JustBeforeEach(func() {
					client.logger = log.NewNoop()
					cb.ExecuteReturns(http.StatusOK, nil)
					client.SetCircuitBreaker(cb)
				})
				It("circuit breaker succeeds", func() {
					statusCode, err := client.Do(ctx, "GET", nil)
					Expect(err).To(BeNil())
					Expect(statusCode).To(Equal(http.StatusOK))
				})
			})
		})
		Context("sad paths", func() {
			Context("missing endpoint", func() {
				JustBeforeEach(func() {
					client.endpoint = nil
				})
				It("returns an error", func() {
					statusCode, err := client.Do(ctx, "GET", nil)
					Expect(err).ToNot(BeNil())
					Expect(statusCode).To(Equal(http.StatusInternalServerError))
				})
			})
			Context("circuit breaker", func() {
				JustBeforeEach(func() {
					client.logger = log.NewNoop()
					cb.ExecuteReturns(nil, errors.New("FAIL"))
					client.SetCircuitBreaker(cb)
				})
				It("circuit breaker fails", func() {
					statusCode, _ := client.Do(ctx, "GET", nil)
					Expect(statusCode).To(Equal(http.StatusFailedDependency))
				})
			})
		})
	})
	// endregion
})
