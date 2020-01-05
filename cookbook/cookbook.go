package cookbook

import (
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr/v2"
	homedir "github.com/mitchellh/go-homedir"
	cb "github.com/appbricks/cloud-builder/cookbook"
)

// Retrieves the embedded cookbook and
// extracts it to a temporary workspace.
// If the cookbook structure is not valid
// then an error will be returned.
func NewCookbook() (*cb.Cookbook, error) {

	var (
		err error

		homeDir       string
		workspacePath string

		cookbook *cb.Cookbook
	)

	if homeDir, err = homedir.Dir(); err != nil {
		return nil, err
	}

	workspacePath = filepath.Join(homeDir, ".cb")
	if err = os.MkdirAll(workspacePath, os.ModePerm); err != nil {
		return nil, err
	}

	box := packr.New("cookbook", "./dist")
	if cookbook, err = cb.NewCookbook(box, workspacePath, os.Stdout, os.Stderr); err != nil {
		return nil, err
	}

	return cookbook, nil
}
