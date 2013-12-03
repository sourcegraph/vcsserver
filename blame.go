package vcsserver

import (
	"encoding/json"
	"github.com/sourcegraph/go-blame/blame"
	"github.com/sourcegraph/go-vcs"
	"log"
	"net/http"
	"os"
	"time"
)

type Author struct {
	Email string
	Name  string
}

type Commit struct {
	CommitID   string
	AuthorDate time.Time
	Author     Author
	Message    string
}

type Hunk struct {
	CommitID string
	File     string
	Start    int
	End      int
}

type BlameResponse struct {
	Commits []*Commit
	Hunks   []*Hunk
}

func init() {
	blame.Log = log.New(os.Stderr, "blame: ", log.LstdFlags)
}

func blameRepository(w http.ResponseWriter, r *http.Request, vcs_ vcs.VCS, dir string) *httpError {
	v := r.URL.Query().Get("v")

	var data BlameResponse
	commits, hunks, err := doBlameRepository(dir, v)
	if err != nil {
		log.Print(err)
		return &httpError{"failed to blame repository", http.StatusInternalServerError}
	}
	data.Commits = commits
	data.Hunks = hunks

	w.Header().Add("content-type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Print(err)
		// too late to return an HTTP error
	}

	return nil
}

var blameIgnores = []string{
	"node_modules", "bower_components",
	"doc", "docs", "build", "vendor",
	".min.js", "-min.js", ".optimized.js", "-optimized.js",
	"dist", "assets",
}

func doBlameRepository(dir, v string) ([]*Commit, []*Hunk, error) {
	hunkMap, commitMap, err := blame.BlameRepository(dir, v, blameIgnores)
	if err != nil {
		return nil, nil, err
	}

	commits := make([]*Commit, len(commitMap))
	i := 0
	for _, commit := range commitMap {
		commits[i] = &Commit{
			CommitID:   commit.ID,
			AuthorDate: commit.AuthorDate,
			Author:     Author{Name: commit.Author.Name, Email: commit.Author.Email},
			Message:    commit.Message,
		}
		i++
	}

	hunks := make([]*Hunk, 0)
	for file, fileHunks := range hunkMap {
		for _, hunk := range fileHunks {
			hunks = append(hunks, &Hunk{
				CommitID: hunk.CommitID,
				File:     file,
				Start:    hunk.CharStart,
				End:      hunk.CharEnd,
			})
		}
	}

	return commits, hunks, nil
}
