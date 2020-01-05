package recipe_test

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mevansam/goutils/logger"
	"github.com/appbricks/cloud-builder/test/data"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRecipe(t *testing.T) {

	var (
		err error

		cookbookPath string
	)

	logger.Initialize()

	_, filename, _, _ := runtime.Caller(0)
	cookbookPath, err = filepath.Abs(fmt.Sprintf("%s/../../cookbook", path.Dir(filename)))
	if err != nil {
		ginkgo.Fail(err.Error())
	}

	err = data.EnsureCookbookIsBuilt(cookbookPath)
	if err != nil {
		ginkgo.Fail(err.Error())
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "recipe")
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
