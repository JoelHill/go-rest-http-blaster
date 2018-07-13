package blaster

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PackageScope", func() {
	var (
		ctx                       context.Context
		defaults                  *Defaults
		logBytes                  []byte
		logBuffer                 *bytes.Buffer
		contextLoggerProviderFunc func(ctx context.Context) (*logrus.Entry, bool)
		requestIDProviderFunc     func(ctx context.Context) (string, bool)
		requestSourceProviderFunc func(ctx context.Context) (string, bool)
		tracerProviderFunc        func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span)
		span                      opentracing.Span
	)

	BeforeEach(func() {
		// zero all package vars
		pkgServiceName = ""
		pkgRequestIDProviderFunc = nil
		pkgRequestSourceProviderFunc = nil
		pkgUserAgent = ""
		pkgStrictREQ014 = false
		pkgStatsdRate = 0
		pkgTracerProviderFunc = nil

		ctx = context.Background()
		logBytes = []byte{}
		logBuffer = bytes.NewBuffer(logBytes)
		span = opentracing.StartSpan("test")

		// default funcs
		contextLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
			logger := logrus.New()
			logger.Out = logBuffer
			return logrus.NewEntry(logger), true
		}
		requestIDProviderFunc = func(ctx context.Context) (string, bool) {
			return "1234", true
		}
		requestSourceProviderFunc = func(ctx context.Context) (string, bool) {
			return "unit-test", true
		}
		tracerProviderFunc = func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span) {
			return r, span
		}

		// defaults struct
		defaults = &Defaults{
			ServiceName:               "unit-test",
			StatsdRate:                1,
			StrictREQ014:              true,
			UserAgent:                 "unit-test",
			RequestIDProviderFunc:     requestIDProviderFunc,
			RequestSourceProviderFunc: requestSourceProviderFunc,
			TracerProviderFunc:        tracerProviderFunc,
		}

		// env
		os.Setenv("SERVICE_NAME", "unit-test")
	})

	// region SETDEFAULTS
	var _ = Describe("Set defaults", func() {
		Context("with all defaults set", func() {
			BeforeEach(func() {
				SetDefaults(defaults)
			})
			It("sets service name", func() {
				Expect(pkgServiceName).To(Equal("unit-test"))
			})
			It("sets statsd rate", func() {
				Expect(pkgStatsdRate).To(Equal(float64(1)))
			})
			It("sets req014 to true", func() {
				Expect(pkgStrictREQ014).To(BeTrue())
			})
			It("sets user agent", func() {
				Expect(pkgUserAgent).To(Equal("unit-test"))
			})
			It("sets request id provider", func() {
				Expect(pkgRequestIDProviderFunc).ToNot(BeNil())

				By("testing request id provider")
				requestID, ok := pkgRequestIDProviderFunc(ctx)
				Expect(requestID).To(Equal("1234"))
				Expect(ok).To(BeTrue())
			})
			It("sets request source provider", func() {
				Expect(pkgRequestSourceProviderFunc).ToNot(BeNil())

				By("testing request source provider")
				requestSource, ok := pkgRequestSourceProviderFunc(ctx)
				Expect(requestSource).To(Equal("unit-test"))
				Expect(ok).To(BeTrue())
			})
			It("sets tracer provider", func() {
				Expect(pkgTracerProviderFunc).ToNot(BeNil())

				By("testing tracer provider")
				r, _ := http.NewRequest("GET", "http://www.foo.bar", os.Stdin)
				request, tspan := pkgTracerProviderFunc(ctx, "test", r)
				Expect(request).ToNot(BeNil())
				Expect(request).To(Equal(r))
				Expect(tspan).To(Equal(span))
			})
		})
	})
	// endregion

	// region ensurePackageVariables
	var _ = Describe("ensure package variables", func() {
		BeforeEach(func() {
			pkgOnce = sync.Once{}
		})
		Context("service name", func() {
			BeforeEach(func() {
				os.Setenv("SERVICE_NAME", "")
				defaults.ServiceName = ""
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			It("will set service name to hostname env var", func() {
				Expect(pkgServiceName).To(Equal(os.Getenv("HOSTNAME")))
			})
		})
		Context("user agent", func() {
			BeforeEach(func() {
				defaults.UserAgent = ""
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			It("will set user agent", func() {
				userAgent := fmt.Sprintf("%s-%s", pkgServiceName, os.Getenv("HOSTNAME"))
				Expect(pkgUserAgent).To(Equal(userAgent))
			})
		})
		Context("user agent to hostname", func() {
			BeforeEach(func() {
				os.Setenv("SERVICE_NAME", "")
				defaults.ServiceName = ""
				defaults.UserAgent = ""
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			It("will set user agent to host name", func() {
				Expect(pkgUserAgent).To(Equal(os.Getenv("HOSTNAME")))
			})
		})
		Context("new relic txn provider", func() {
			BeforeEach(func() {
				logrus.SetOutput(logBuffer)
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
		})
	})
	// endregion

	// region newHTTPClient
	var _ = Describe("newHTTPClient", func() {
		Context("testing/mocking path", func() {
			It("should return the default http client", func() {
				httpClient := newHTTPClient()
				Expect(httpClient).To(Equal(http.DefaultClient))
			})
		})
		Context("non-testing client", func() {
			BeforeEach(func() {
				os.Setenv("MOCKING_HTTP", "")
			})
			AfterEach(func() {
				os.Setenv("MOCKING_HTTP", "true")
			})
			It("returns the tuned base client", func() {
				httpClient := newHTTPClient()
				Expect(httpClient.Timeout).To(Equal(requestTimeout))
				transport := httpClient.Transport.(*http.Transport)
				Expect(transport.DisableCompression).To(BeFalse())
				Expect(transport.DisableKeepAlives).To(BeFalse())
				Expect(transport.ExpectContinueTimeout).To(Equal(1 * time.Second))
				Expect(transport.MaxIdleConnsPerHost).To(Equal(maxIdleConnsPerHost))
				Expect(transport.MaxIdleConns).To(Equal(maxIdleConns))
				Expect(transport.IdleConnTimeout).To(Equal(idleTimeout))
				Expect(transport.TLSHandshakeTimeout).To(Equal(tlsTimeout))
			})
		})
	})
	// endregion

	// region NewClient
	var _ = Describe("NewClient", func() {
		Context("happy path", func() {
			BeforeEach(func() {
				SetDefaults(defaults)
			})
			It("gets the base client", func() {
				httpClient, err := NewClient("http://www.invisionapp.com")
				Expect(err).To(BeNil())
				Expect(httpClient).ToNot(BeNil())
				Expect(httpClient.headers[userAgentHeader]).To(Equal(pkgUserAgent))
				Expect(httpClient.headers[contentTypeHeader]).To(Equal(jsonType))
				Expect(httpClient.headers[callingServiceHeader]).To(Equal(pkgServiceName))
				Expect(httpClient.headers[acceptHeader]).To(Equal(jsonType))
			})
		})
		Context("sad path", func() {
			BeforeEach(func() {
				SetDefaults(defaults)
			})
			It("throws an error", func() {
				_, err := NewClient("")
				Expect(err).ToNot(BeNil())
			})
		})
	})
	// endregion
})
