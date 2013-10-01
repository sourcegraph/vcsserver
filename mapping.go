package vcsserver

import (
	"errors"
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"net/http/cgi"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GitHTTPBackend is the path to the git-http-backend executable.
var GitHTTPBackend string = "/usr/lib/git-core/git-http-backend"

// Mapping represents a mapping from a request path on this server to an origin
// VCS repository URL.
type Mapping struct {
	// Host is the hostname of the remote server to use when constructing the
	// origin URL.
	Host string

	// VCS is the type of version control system (e.g., Git or Hg).
	VCS vcs.VCS

	// Repo is a regular expression that matches the repository name.
	Repo *regexp.Regexp

	// Scheme is the URL scheme to use when constructing the origin URL.
	Scheme string
}

// ServeHTTP implements net/http.Handler.
func (m Mapping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Match repo in route.
	repo := m.Repo.FindString(strings.TrimPrefix(r.URL.Path, "/"+m.Host))
	if repo == "" {
		err := errors.New("invalid repo")
		log.Print(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dir := repoDir(m.Host, m.VCS, repo)

	// Find or create repo dir.
	fi, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		log.Print(err)
		http.Error(w, "error opening repo directory", http.StatusInternalServerError)
		return
	}
	if fi != nil && !fi.IsDir() {
		err = errors.New("repo path is not directory")
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	remoteURL := (&url.URL{
		Scheme: m.Scheme,
		Host:   m.Host,
		Path:   repo,
	}).String()

	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(dir), 0700)
		if err != nil {
			log.Print(err)
			http.Error(w, "error creating repo parent directory", http.StatusInternalServerError)
			return
		}

		log.Printf("cloning mirror in %s", dir)
		err = m.VCS.CloneMirror(remoteURL, dir)
		if err != nil {
			log.Print(err)
			http.Error(w, "error cloning mirror", http.StatusInternalServerError)
			return
		}
	} else if r.Header.Get("pragma") == "no-cache" {
		log.Printf("updating mirror in %s", dir)
		err = m.VCS.UpdateMirror(dir)
		if err != nil {
			log.Print(err)
			http.Error(w, "error updating mirror", http.StatusInternalServerError)
			return
		}
	}

	backend := &cgi.Handler{
		Path: GitHTTPBackend,
		Dir:  dir,
		Env:  []string{"GIT_HTTP_EXPORT_ALL=", "GIT_PROJECT_ROOT=" + filepath.Join(StorageDir, m.VCS.ShortName())},
	}
	backend.ServeHTTP(w, r)
}
