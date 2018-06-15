package cbapiclient

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/InVisionApp/cbapiclient/fakes"
	"github.com/newrelic/go-agent"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		Dogs []Dog `json:"cats"`
	}

	var (
		client         *Client
		cb             *fakes.FakeCircuitBreakerPrototype
		statsd         *fakes.FakeStatsdClientPrototype
		ctx            context.Context
		endpoint       *url.URL
		logBytes       []byte
		logBuffer      *bytes.Buffer
		byteLoggerFunc func(ctx context.Context) (*logrus.Entry, bool)
		petStore       *PetStore
		petStoreStr    string
		endpointStr    string
	)

	BeforeEach(func() {
		ctx = context.Background()

		pkgRequestIDProviderFunc = func(ctx context.Context) (string, bool) {
			return "unit-test-request-id", true
		}
		pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
			return "unit-test-request-source", true
		}
		pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
			logger := logrus.New()
			logger.Out = ioutil.Discard
			return logrus.NewEntry(logger), true
		}
		pkgNRTxnProviderFunc = func(ctx context.Context) (newrelic.Transaction, bool) {
			return nil, false
		}

		pkgStrictREQ014 = true
		pkgStatsdFailureTag = "success"
		pkgStatsdFailureTag = "failure"
		pkgUserAgent = "unit test"
		pkgServiceName = "unit test"
		endpointStr = "http://www.invisionapp.com"

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
		petStoreStr = `{"cats":[{"name":"Scruffy","color":"Orange"},{"name":"Shadow","color":"Black"}],"dogs":[{"name":"Lulu","breed":"Basset"},{"name":"Pippy","breed":"Jack Russell Terrier"}]}`

		cb.ExecuteReturns(200, nil)
		statsd.IncrReturns(nil)
		statsd.TimingReturns(nil)
	})

	JustBeforeEach(func() {
		client = &Client{
			endpoint:     endpoint,
			client:       http.DefaultClient,
			statsdClient: statsd,
			statsdStat:   "fake-api-call",
			statsdTags:   []string{"foo", "bar"},
			headers: map[string]string{
				userAgentHeader:      pkgUserAgent,
				contentTypeHeader:    jsonType,
				callingServiceHeader: pkgServiceName,
				acceptHeader:         jsonType,
			},
		}
	})

	// region REQ014
	var _ = Describe("REQ014 checks", func() {
		Context("happy path", func() {
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(petStoreStr)

				_, err := client.Get(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("happy path - request source header missing, not required", func() {
			JustBeforeEach(func() {
				pkgStrictREQ014 = false
				pkgRequestSourceProviderFunc = func(ctx context.Context) (string, bool) {
					return "", false
				}
			})
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(petStoreStr)

				_, err := client.Get(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("happy path - calling service header missing, not required", func() {
			JustBeforeEach(func() {
				pkgStrictREQ014 = false
				delete(client.headers, callingServiceHeader)
			})
			It("throws no errors", func() {
				defer gock.Off()
				gock.New(endpointStr).Get("/").Reply(200).BodyString(petStoreStr)

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

})
