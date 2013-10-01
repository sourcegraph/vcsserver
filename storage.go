package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"path/filepath"
)

var StorageDir string = "/tmp/vcsserver"

func repoDir(host string, VCS vcs.VCS, path string) string {
	return filepath.Join(StorageDir, VCS.ShortName(), host, path)
}
