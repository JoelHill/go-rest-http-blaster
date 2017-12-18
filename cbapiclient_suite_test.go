package cbapiclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCbapiclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cbapiclient Suite")
}
