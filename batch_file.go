package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"os"
	"strings"
)

const returnFirstExist = "first-exist"

func batchFile(w http.ResponseWriter, r *http.Request, vcs vcs.VCS, dir string, extraPath string) *httpError {
	rev := strings.TrimPrefix(extraPath, "/v-batch/")

	q := r.URL.Query()
	returns := q.Get("return")
	filepaths := q["file"]

	if returns != returnFirstExist {
		return &httpError{"unrecognized ?returns param", http.StatusBadRequest}
	}

	if len(filepaths) == 0 {
		return &httpError{"no files specified", http.StatusBadRequest}
	}

	v, err := vcs.Open(dir)
	if err != nil {
		log.Print(err)
		return &httpError{"failed to open repository", http.StatusInternalServerError}
	}

	for _, path := range filepaths {
		data, err := v.ReadFileAtRevision(path, rev)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			log.Print(err)
			return &httpError{"failed to read file at revision", http.StatusInternalServerError}
		}
		w.Write(data)
		return nil
	}

	return &httpError{"not found", http.StatusNotFound}
}
