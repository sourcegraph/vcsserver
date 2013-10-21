package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"path/filepath"
)

func proxy(w http.ResponseWriter, r *http.Request, route *route, dir string) *httpError {
	var backend *cgi.Handler
	logger := log.New(os.Stderr, "proxy "+route.uri+": ", 0)
	switch route.vcs {
	case vcs.Git:
		r.URL.Path = "/" + route.uri + route.extraPath
		backend = &cgi.Handler{
			Path:   GitHTTPBackend,
			Dir:    dir,
			Env:    []string{"GIT_HTTP_EXPORT_ALL=", "GIT_PROJECT_ROOT=" + filepath.Join(StorageDir, route.vcs.ShortName())},
			Logger: logger,
		}
	case vcs.Hg:
		rootPath, err := filepath.Rel(StorageDir, dir)
		if err != nil {
			log.Print(err)
			return &httpError{"failed to get root path", http.StatusInternalServerError}
		}
		r.URL.Path = route.extraPath
		backend = &cgi.Handler{
			Path: Python27,
			Root: "/" + rootPath,
			Dir:  dir,
			Env:  []string{"HG_REPO_DIR=" + dir},
			// condensed hgweb.cgi script
			Args:   []string{"-c", "import os;from mercurial import demandimport;demandimport.enable();from mercurial.hgweb import hgweb,wsgicgi;application=hgweb(os.getenv('HG_REPO_DIR'));wsgicgi.launch(application)"},
			Logger: logger,
		}
	default:
		return &httpError{"unknown VCS type", http.StatusBadRequest}
	}
	backend.ServeHTTP(w, r)
	return nil
}
