package vcsserver

import (
	"errors"
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

func (h *Handler) startCloneOrUpdate(dir string) (c chan *httpError, shouldWait bool) {
	h.currentlyUpdatingLock.Lock()
	defer h.currentlyUpdatingLock.Unlock()
	_, present := h.currentlyUpdating[dir]
	if !present {
		h.currentlyUpdating[dir] = make([]chan *httpError, 0)
		return nil, false
	}
	c = make(chan *httpError, 0)
	h.currentlyUpdating[dir] = append(h.currentlyUpdating[dir], c)
	return c, true
}

func (h *Handler) endCloneOrUpdate(dir string, herr *httpError) {
	h.currentlyUpdatingLock.Lock()
	defer h.currentlyUpdatingLock.Unlock()
	waiters := h.currentlyUpdating[dir]
	for _, c := range waiters {
		c <- herr
	}
	delete(h.currentlyUpdating, dir)
}

func (h *Handler) cloneOrUpdate(vcs vcs.VCS, dir string, cloneURL string, forceUpdate bool) (herr *httpError) {
	if Offline {
		log.Printf("Skipping cloneOrUpdate of %s in offline mode", cloneURL)
		return nil
	}

	c, shouldWait := h.startCloneOrUpdate(dir)
	if shouldWait {
		err := <-c
		if err != nil {
			err = &httpError{message: err.message, statusCode: err.statusCode}
			err.message = "after waiting: " + err.message
		}
		return err
	}

	mu := h.ensureRepoMutex(dir)
	mu.Lock()
	defer mu.Unlock()

	defer func() {
		h.endCloneOrUpdate(dir, herr)
	}()

	// Find or create repo dir.
	fi, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		log.Print(err)
		return &httpError{"error opening repo directory", http.StatusInternalServerError}
	}
	if fi != nil && !fi.IsDir() {
		err = errors.New("repo path is not directory")
		log.Print(err)
		return &httpError{err.Error(), http.StatusInternalServerError}
	}

	// Clone if it doesn't exist yet. If it exists, only update if forceUpdate.
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(dir), 0700)
		if err != nil {
			log.Print(err)
			return &httpError{"error creating repo parent directory", http.StatusInternalServerError}
		}

		record("clone", cloneURL)
		err = vcs.CloneMirror(cloneURL, dir)
		if err != nil {
			log.Print(err)
			return &httpError{"error cloning mirror", http.StatusInternalServerError}
		}
	} else if forceUpdate {
		record("update", cloneURL)
		err = vcs.UpdateMirror(dir)
		if err != nil {
			log.Print(err)
			return &httpError{"error updating mirror", http.StatusInternalServerError}
		}
	}

	return nil
}

// record records an action that occurred. It currently is only used for testing
// (to ensure that specific actions occurred), but it could be used for tracking
// statistics in the future.
func record(action, cloneURL string) {
	log.Print(action + ":" + cloneURL)
	actions[action+":"+cloneURL]++
}

var actions = make(map[string]uint)
