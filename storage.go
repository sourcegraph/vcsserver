package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"path/filepath"
)

// StorageDir is the root directory underneath which repositories are stored.
var StorageDir = "/tmp/vcsserver"

func repoDir(vcs vcs.VCS, uri string) string {
	return filepath.Join(StorageDir, vcs.ShortName(), uri)
}
