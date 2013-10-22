package vcsserver

import (
	"bufio"
	"github.com/sourcegraph/go-cgi/cgi"
	"github.com/sourcegraph/go-vcs"
	"log"
	"net"
	"net/http"
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

	rr := newRecorder(w)
	backend.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		log.Printf("CGI: HTTP response code %d", rr.Code)
	}

	return nil
}

// responseRecorder is an implementation of http.ResponseWriter that
// records its HTTP status code and body length.
type responseRecorder struct {
	Code       int // the HTTP response code from WriteHeader
	BodyLength int

	underlying http.ResponseWriter
}

// newRecorder returns an initialized ResponseRecorder.
func newRecorder(underlying http.ResponseWriter) *responseRecorder {
	return &responseRecorder{underlying: underlying}
}

// Header returns the header map from the underlying ResponseWriter.
func (rw *responseRecorder) Header() http.Header {
	return rw.underlying.Header()
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *responseRecorder) Write(buf []byte) (int, error) {
	rw.BodyLength += len(buf)
	if rw.Code == 0 {
		rw.Code = http.StatusOK
	}
	return rw.underlying.Write(buf)
}

// WriteHeader sets rw.Code.
func (rw *responseRecorder) WriteHeader(code int) {
	rw.Code = code
	rw.underlying.WriteHeader(code)
}

func (rw *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return rw.underlying.(http.Hijacker).Hijack()
}
