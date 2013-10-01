package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type mappingTest struct {
	mappings        map[string]Mapping
	vcs             vcs.VCS
	cloneURLPath    string
	ensureFileLocal string
}

func TestMapping(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	hg, git := vcs.VCSByName["hg"], vcs.VCSByName["git"]

	tests := []mappingTest{
		{
			mappings:        map[string]Mapping{"/github.com/": {"github.com", git, regexp.MustCompile("^/([^/]+)/([^/])+"), "git"}},
			vcs:             git,
			cloneURLPath:    "/github.com/sourcegraph/nodejs-sample.git",
			ensureFileLocal: "package.json",
		},
		{
			mappings:        map[string]Mapping{"/bitbucket.org/": {"bitbucket.org", hg, regexp.MustCompile("^/([^/]+)/([^/])+"), "https"}},
			vcs:             hg,
			cloneURLPath:    "/bitbucket.org/sqs/go-vcs-hgtest",
			ensureFileLocal: "foo",
		},
	}

	for i, test := range tests {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}

		testMapping(t, test)
	}
}

func testMapping(t *testing.T, test mappingTest) {
	mux := http.NewServeMux()
	for path, handler := range test.mappings {
		mux.Handle(path, handler)
	}
	s := httptest.NewServer(mux)
	defer s.Close()

	// Make a temp dir for the client to clone the repo into.
	tmpdir, err := ioutil.TempDir("", "vcsserver-local")
	if err != nil {
		t.Fatal("TempDir failed:", err)
	}
	defer os.RemoveAll(tmpdir)

	localRepoDir := filepath.Join(tmpdir, "repo")
	_, err = test.vcs.Clone(s.URL+test.cloneURLPath, localRepoDir)
	if err != nil {
		t.Fatal("Clone failed:", err)
	}

	var f string
	switch test.vcs {
	case vcs.Git:
		f = filepath.Join(localRepoDir, ".git/config")
	case vcs.Hg:
		f = filepath.Join(localRepoDir, ".hg/hgrc")
	default:
		t.Fatal("unhandled VCS type")
	}
	if !isFile(f) {
		t.Errorf("want file %s to exist", f)
	}

	if f := filepath.Join(localRepoDir, test.ensureFileLocal); !isFile(f) {
		t.Errorf("want file %s to exist", f)
	}

	var ok bool
	for _, m := range test.mappings {
		storedRepoDir := repoDir(m.Host, m.VCS, strings.TrimPrefix(test.cloneURLPath, "/"+m.Host+"/"))
		var f string
		switch m.VCS {
		case vcs.Git:
			f = filepath.Join(storedRepoDir, "config")
		case vcs.Hg:
			f = filepath.Join(storedRepoDir, ".hg/hgrc")
		default:
			t.Fatal("unhandled VCS type")
		}
		if isFile(f) {
			ok = true
		}
	}
	if !ok {
		t.Errorf("no storedRepoDir contains a cloned repo (did the repo get cloned by the mapping handler?)")
	}
}

// isFile returns true if path is an existing directory, and false otherwise.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
