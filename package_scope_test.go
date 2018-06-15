package cbapiclient

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/InVisionApp/cbapiclient/fakes"
	"github.com/newrelic/go-agent"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PackageScope", func() {
	var (
		ctx                             context.Context
		defaults                        *Defaults
		logBytes                        []byte
		logBuffer                       *bytes.Buffer
		contextLoggerProviderFunc       func(ctx context.Context) (*logrus.Entry, bool)
		requestIDProviderFunc           func(ctx context.Context) (string, bool)
		requestSourceProviderFunc       func(ctx context.Context) (string, bool)
		tracerProviderFunc              func(ctx context.Context, operationName string, r *http.Request) (*http.Request, opentracing.Span)
		newRelicTransactionProviderFunc func(ctx context.Context) (newrelic.Transaction, bool)
		span                            opentracing.Span
		nrtx                            *fakes.FakeTransaction
	)

	BeforeEach(func() {
		// zero all package vars
		pkgServiceName = ""
		pkgNRTxnProviderFunc = nil
		pkgCtxLoggerProviderFunc = nil
		pkgRequestIDProviderFunc = nil
		pkgRequestSourceProviderFunc = nil
		pkgUserAgent = ""
		pkgStrictREQ014 = false
		pkgStatsdRate = 0
		pkgStatsdSuccessTag = ""
		pkgStatsdFailureTag = ""
		pkgTracerProviderFunc = nil

		ctx = context.Background()
		logBytes = []byte{}
		logBuffer = bytes.NewBuffer(logBytes)
		span = opentracing.StartSpan("test")
		nrtx = &fakes.FakeTransaction{}

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
		newRelicTransactionProviderFunc = func(ctx context.Context) (newrelic.Transaction, bool) {
			return nrtx, true
		}

		// defaults struct
		defaults = &Defaults{
			ServiceName:                     "unit-test",
			ContextLoggerProviderFunc:       contextLoggerProviderFunc,
			StatsdRate:                      1,
			StatsdFailureTag:                "failed",
			StatsdSuccessTag:                "success",
			StrictREQ014:                    true,
			UserAgent:                       "unit-test",
			RequestIDProviderFunc:           requestIDProviderFunc,
			RequestSourceProviderFunc:       requestSourceProviderFunc,
			TracerProviderFunc:              tracerProviderFunc,
			NewRelicTransactionProviderFunc: newRelicTransactionProviderFunc,
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
			It("sets context log provider", func() {
				Expect(pkgCtxLoggerProviderFunc).ToNot(BeNil())

				By("testing context log provider")
				logger, ok := pkgCtxLoggerProviderFunc(ctx)
				Expect(logger).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
			It("sets statsd rate", func() {
				Expect(pkgStatsdRate).To(Equal(float64(1)))
			})
			It("sets statsd failure flag", func() {
				Expect(pkgStatsdFailureTag).To(Equal("failed"))
			})
			It("sets statsd success flag", func() {
				Expect(pkgStatsdSuccessTag).To(Equal("success"))
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
			It("sets mew relic transaction provider", func() {
				Expect(pkgNRTxnProviderFunc).ToNot(BeNil())

				By("testing new relic transaction provider")
				txn, ok := pkgNRTxnProviderFunc(ctx)
				Expect(txn).To(Equal(nrtx))
				Expect(ok).To(BeTrue())
			})
		})
	})
	// endregion

	// region ensurePackageVariables
	var _ = Describe("ensure package variables", func() {
		BeforeEach(func() {
			pkgOnce = sync.Once{}
		})
		Context("context logger", func() {
			BeforeEach(func() {
				logrus.SetOutput(logBuffer)
				defaults.ContextLoggerProviderFunc = nil
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will warn about missing context logger provider", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: No ContextLoggerProviderFunc default set.  A new logger will be used for each request"))
				Expect(pkgCtxLoggerProviderFunc).ToNot(BeNil())

				By("testing default context logger provider")
				logger, ok := pkgCtxLoggerProviderFunc(ctx)
				Expect(logger).ToNot(BeNil())
				Expect(ok).To(BeTrue())
			})
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
				defaults.NewRelicTransactionProviderFunc = nil
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will warn about missing context logger provider", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: no NewRelicTransactionProviderFunc set"))
				Expect(pkgNRTxnProviderFunc).ToNot(BeNil())

				By("testing default newrelic txn provider")
				txn, ok := pkgNRTxnProviderFunc(ctx)
				Expect(txn).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})
		Context("request id", func() {
			BeforeEach(func() {
				logrus.SetOutput(logBuffer)
				defaults.RequestIDProviderFunc = nil
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will warn about missing request id provider", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: No RequestIDProviderFunc default set.  The Request-ID header will be absent in each request unless set manually"))
				Expect(pkgRequestIDProviderFunc).ToNot(BeNil())

				By("testing request id provider")
				requestID, ok := pkgRequestIDProviderFunc(ctx)
				Expect(requestID).To(Equal(""))
				Expect(ok).To(BeTrue())
			})
		})
		Context("request source", func() {
			BeforeEach(func() {
				logrus.SetOutput(logBuffer)
				defaults.RequestSourceProviderFunc = nil
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will warn about missing request source provider", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: No RequestSourceProviderFunc default set.  The Request-Source header will be absent in each request unless set manually"))
				Expect(pkgRequestSourceProviderFunc).ToNot(BeNil())

				By("testing request source provider")
				requestSource, ok := pkgRequestSourceProviderFunc(ctx)
				Expect(requestSource).To(Equal(""))
				Expect(ok).To(BeFalse())
			})
		})
		Context("statsd failure tag", func() {
			BeforeEach(func() {
				pkgStatsdFailureTag = ""
				logrus.SetOutput(logBuffer)
				defaults.StatsdFailureTag = ""
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will report about using default statsd failure tag", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: no statsd failure tag provided.  using processed:failure."))
				Expect(pkgStatsdFailureTag).To(Equal("processed:failure"))
			})
		})
		Context("statsd success tag", func() {
			BeforeEach(func() {
				pkgStatsdSuccessTag = ""
				logrus.SetOutput(logBuffer)
				defaults.StatsdSuccessTag = ""
				SetDefaults(defaults)
				ensurePackageVariables()
			})
			AfterEach(func() {
				logrus.SetOutput(os.Stderr)
			})
			It("will report about using default statsd success tag", func() {
				Expect(string(logBuffer.Bytes())).To(ContainSubstring("cbapiclient: no statsd success tag provided.  using processed:success."))
				Expect(pkgStatsdSuccessTag).To(Equal("processed:success"))
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
