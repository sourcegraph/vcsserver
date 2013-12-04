package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Handler contains settings for vcsserver and implements http.Handler.
type Handler struct {
	// Hosts is a whitelist of hosts whose repositories may be accessed.
	Hosts []string

	currentlyUpdatingLock sync.Mutex
	currentlyUpdating     map[string][]chan *httpError

	repoAccessLock sync.Mutex
	repoAccess     map[string]*sync.Mutex
}

func New(hosts []string) *Handler {
	enableBlameLog()
	return &Handler{
		Hosts:             hosts,
		currentlyUpdating: make(map[string][]chan *httpError),
		repoAccess:        make(map[string]*sync.Mutex),
	}
}

// Router constructs a handler that provides cloning and file access.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := router(h.Hosts, r.URL.Path)
	if err != nil {
		http.Error(w, err.message, err.statusCode)
		return
	}

	// If this is the first op in a transaction, then update the repo
	// from the remote. (We don't want to try to update it for each op in a
	// transaction.)
	var forceUpdate bool
	if r.Header.Get("pragma") == "no-cache" {
		// git clone/fetch/pull set `Pragma: no-cache` on its initial request.
		forceUpdate = true
	}
	if route.vcs == vcs.Hg && r.URL.Query().Get("cmd") == "capabilities" {
		// hg clone/pull's initial request query string contains
		// `cmd=capabilities`.
		forceUpdate = true
	}

	// Clone or update the requested repo.
	dir := repoDir(route.vcs, route.uri)
	err = h.cloneOrUpdate(route.vcs, dir, route.cloneURL, forceUpdate)
	if err != nil {
		http.Error(w, err.message, err.statusCode)
		return
	}

	mu := h.ensureRepoMutex(dir)
	mu.Lock()
	defer mu.Unlock()

	switch route.action {
	case proxyAction:
		err = proxy(w, r, route, dir)
	case singleFileAction:
		err = file(w, r, route.vcs, dir, route.extraPath)
	case blameAction:
		err = blameRepository(w, r, route.vcs, dir)
	default:
		panic("unknown action: " + string(route.action))
	}

	if err != nil {
		http.Error(w, err.message, err.statusCode)
	}
}

func (h *Handler) ensureRepoMutex(dir string) *sync.Mutex {
	h.repoAccessLock.Lock()
	defer h.repoAccessLock.Unlock()
	_, present := h.repoAccess[dir]
	if !present {
		h.repoAccess[dir] = new(sync.Mutex)
	}
	return h.repoAccess[dir]
}

type action string

const (
	proxyAction      action = "proxy"
	singleFileAction        = "singleFile"
	blameAction             = "blame"
)

type httpError struct {
	message    string
	statusCode int
}

type route struct {
	vcs           vcs.VCS
	cloneURL, uri string
	action        action
	extraPath     string
}

var pathPattern = regexp.MustCompile(`^/(?P<pathComponents>\d+)/(?P<vcs>git|hg)/(?P<scheme>http|https|git)/(?P<host>[a-zA-Z0-9.-]+)/(?P<path>.*)$`)

func router(hosts []string, path string) (*route, *httpError) {
	m := pathPattern.FindStringSubmatch(path)
	if m == nil {
		return nil, &httpError{"bad path", http.StatusNotFound}
	}

	numPathComponents, err := strconv.Atoi(m[1])
	if err != nil {
		return nil, &httpError{"first path component must be number of path components in repo", http.StatusBadRequest}
	}

	vcsName, scheme, host, path := m[2], m[3], m[4], m[5]

	// Check that specified host is in list of allowable hosts.
	hostOK := false
	for _, okHost := range hosts {
		if okHost == host {
			hostOK = true
			break
		}
	}
	if !hostOK {
		return nil, &httpError{"access to specified host is not allowed", http.StatusForbidden}
	}

	cloneURL := &url.URL{
		Scheme: scheme,
		Host:   strings.ToLower(host),
	}
	repoPath, extraPath := bisectBeforeNth(path, "/", numPathComponents)
	cloneURL.Path = "/" + filepath.Clean(repoPath)
	uri := cloneURL.Host + cloneURL.Path

	var action action
	if strings.HasPrefix(extraPath, "/v/") {
		action = singleFileAction
	} else if strings.HasPrefix(extraPath, "/api/blame") {
		action = blameAction
	} else {
		action = proxyAction
	}

	return &route{
		vcs:       vcs.VCSByName[vcsName],
		cloneURL:  cloneURL.String(),
		uri:       uri,
		action:    action,
		extraPath: extraPath,
	}, nil
}

// bisectBeforeNth splits s into 2 strings on the nth occurrence of sep.
func bisectBeforeNth(s string, sep string, n int) (string, string) {
	seen := 0
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			seen++
		}
		if seen == n {
			return s[:i], s[i:]
		}
	}
	return s, ""
}
