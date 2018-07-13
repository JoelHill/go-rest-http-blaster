package blaster

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCbapiclient(t *testing.T) {
	os.Setenv("MOCKING_HTTP", "true")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cbapiclient Suite")
}
