package vcsserver

import (
	"github.com/sourcegraph/go-cgi/cgi"
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"net/http/httptest"
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

	// I don't know why, but having the CGI process write to a ResponseRecorder
	// makes it die less often (never?). To test this, run:
	//   go test -test.run=Concurrent -test.v
	// until it fails. It shouldn't fail.
	rw := httptest.NewRecorder()
	backend.ServeHTTP(rw, r)
	rw.Flush()

	for k, vs := range rw.Header() {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(rw.Code)
	w.Write(rw.Body.Bytes())

	return nil
}
