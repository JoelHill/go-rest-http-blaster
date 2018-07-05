package cbapiclient

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Logrus Shim", func() {
	var testContext context.Context
	BeforeEach(func() {
		testContext = context.Background()
	})
	Context("with logger provider set", func() {
		BeforeEach(func() {
			pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
				return logrus.NewEntry(logrus.New()), true
			}
		})
		It("creates a logger", func() {
			lg := logrusShim(testContext)
			Expect(lg).ToNot(BeNil())
		})
	})
	Context("no logger provider set", func() {
		BeforeEach(func() {
			pkgCtxLoggerProviderFunc = func(ctx context.Context) (*logrus.Entry, bool) {
				return logrus.NewEntry(logrus.New()), false
			}
		})
		It("creates a default logger", func() {
			lg := logrusShim(testContext)
			Expect(lg).ToNot(BeNil())
		})
	})
})
