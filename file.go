package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"os"
	"strings"
)

func file(w http.ResponseWriter, r *http.Request, vc vcs.VCS, dir string, extraPath string) *httpError {
	extraPath = strings.TrimPrefix(extraPath, "/v/")
	parts := strings.SplitN(extraPath, "/", 2)
	if len(parts) != 2 {
		return &httpError{"bad file path", http.StatusNotFound}
	}
	rev, path := parts[0], parts[1]
	v, err := vc.Open(dir)
	if err != nil {
		log.Print(err)
		return &httpError{"failed to open repository", http.StatusInternalServerError}
	}

	data, filetype, err := v.ReadFileAtRevision(path, rev)
	if os.IsNotExist(err) {
		return &httpError{"not found", http.StatusNotFound}
	} else if err != nil {
		log.Print(err)
		return &httpError{"failed to read file at revision", http.StatusInternalServerError}
	}
	if filetype == vcs.Dir {
		w.Header().Set("Content-Type", "application/x-directory")
	}
	w.Write(data)
	return nil
}
