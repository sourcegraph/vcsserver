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
	mappings     map[string]Mapping
	vcs          vcs.VCS
	cloneURLPath string
}

func TestMapping(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	git := vcs.VCSByName["git"]

	tests := []mappingTest{
		{
			mappings:     map[string]Mapping{"/github.com/": {"github.com", git, regexp.MustCompile("^/([^/]+)/([^/])+"), "git"}},
			vcs:          git,
			cloneURLPath: "/github.com/sourcegraph/nodejs-sample.git",
		},
	}

	for i, test := range tests {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal(err)
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
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	localRepoDir := filepath.Join(tmpdir, "repo")
	_, err = test.vcs.Clone(s.URL+"/"+test.cloneURLPath, localRepoDir)
	if err != nil {
		t.Fatal(err)
	}

	if f := filepath.Join(localRepoDir, ".git/config"); !isFile(f) {
		t.Errorf("want file %s to exist", f)
	}

	var ok bool
	for _, m := range test.mappings {
		storedRepoDir := repoDir(m.Host, m.VCS, strings.TrimPrefix(test.cloneURLPath, "/"+m.Host+"/"))
		f := filepath.Join(storedRepoDir, "config")
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
