package cookbook_test

import (
	"path"
	"runtime"
	"testing"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/mevansam/goutils/logger"
	"github.com/appbricks/cloud-builder/test/data"
)

func TestCookbook(t *testing.T) {

	var (
		err           error
		sourceDirPath string
	)

	logger.Initialize()

	_, filename, _, _ := runtime.Caller(0)
	sourceDirPath = path.Dir(filename)

	err = data.EnsureCookbookIsBuilt(sourceDirPath)
	if err != nil {
		ginkgo.Fail(err.Error())
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "cookbook")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
