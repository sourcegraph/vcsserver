package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"os"
	"path/filepath"
	"strings"
)

// StorageDir is the root directory underneath which repositories are stored.
var StorageDir = "/tmp/vcsserver"

func repoDir(vcs vcs.VCS, uri string) string {
	preferred := filepath.Join(StorageDir, vcs.ShortName(), uri)

	// if we're running offline, try harder to find a local copy
	if Offline {
		if alternate := strings.TrimSuffix(preferred, ".git"); isDir(alternate) {
			return alternate
		}
	}

	return preferred
}

// IsDir returns true if path is an existing directory, and false otherwise.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
