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
var GitHTTPBackend = os.Getenv("GIT_HTTP_BACKEND")

// Python27 is the path to Python 2.7.
var Python27 = os.Getenv("PYTHON27")

func init() {
	if GitHTTPBackend == "" {
		GitHTTPBackend = "/usr/lib/git-core/git-http-backend"
	}
	if Python27 == "" {
		Python27 = "/usr/bin/python2.7"
	}
}

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

	extrapath := strings.TrimPrefix(r.URL.Path, "/"+m.Host+repo)
	if strings.HasPrefix(extrapath, "/v/") {
		extrapath = strings.TrimPrefix(extrapath, "/v/")
		parts := strings.SplitN(extrapath, "/", 2)
		if len(parts) != 2 {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		rev, path := parts[0], parts[1]
		v, err := m.VCS.Open(dir)
		if err != nil {
			log.Print(err)
			http.Error(w, "failed to open repository", http.StatusInternalServerError)
			return
		}

		data, err := v.ReadFileAtRevision(path, rev)
		if os.IsNotExist(err) {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		} else if err != nil {
			log.Print(err)
			http.Error(w, "failed to read file at revision", http.StatusInternalServerError)
			return
		}
		w.Write(data)
	} else {
		var backend *cgi.Handler
		switch m.VCS {
		case vcs.Git:
			backend = &cgi.Handler{
				Path: GitHTTPBackend,
				Dir:  dir,
				Env:  []string{"GIT_HTTP_EXPORT_ALL=", "GIT_PROJECT_ROOT=" + filepath.Join(StorageDir, m.VCS.ShortName())},
			}
		case vcs.Hg:
			backend = &cgi.Handler{
				Path: Python27,
				Root: "/" + m.Host + repo,
				Dir:  dir,
				Env:  []string{"HG_REPO_DIR=" + dir},
				// condensed hgweb.cgi script
				Args: []string{"-c", "import os;from mercurial import demandimport;demandimport.enable();from mercurial.hgweb import hgweb,wsgicgi;application=hgweb(os.getenv('HG_REPO_DIR'));wsgicgi.launch(application)"},
			}
		default:
			w.WriteHeader(http.StatusNoContent)
			return
		}
		backend.ServeHTTP(w, r)
	}
}
