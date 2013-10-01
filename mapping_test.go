package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"io"
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
	mappings map[string]Mapping
	clones   []cloneTest
	getFiles []getFileTest
}

type cloneTest struct {
	vcs                   vcs.VCS
	url                   string
	ensureLocalFileExists string
}

type getFileTest struct {
	url        string
	statusCode int
	data       string
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
			mappings: map[string]Mapping{"/github.com/": {"github.com", git, regexp.MustCompile("^/([^/]+)/([^/])+"), "git"}},
			clones: []cloneTest{{
				vcs: git,
				url: "/github.com/sqs/vcsserver-gittest.git",
				ensureLocalFileExists: "foo",
			}},
			getFiles: []getFileTest{
				{url: "/github.com/sqs/vcsserver-gittest/v/d3dd4c84e9e429e28e05d53a04651bce084f0565/foo", data: "Hello, foo"},
				{url: "/github.com/sqs/vcsserver-gittest/v/master/foo", data: "Hello, foo!!!"},
				{url: "/github.com/sqs/vcsserver-gittest/v/master/qux", statusCode: http.StatusNotFound},
				{url: "/github.com/sqs/vcsserver-gittest/v/quxbranch/qux", data: "Hello, qux"},
				{url: "/github.com/sqs/vcsserver-gittest/v/quxbranch/foo", data: "Hello, foo!!!"},

				{url: "/github.com/sqs/vcsserver-gittest/v/doesntexist/foo", statusCode: http.StatusNotFound},
			},
		},
		{
			mappings: map[string]Mapping{"/bitbucket.org/": {"bitbucket.org", hg, regexp.MustCompile("^/([^/]+)/([^/])+"), "https"}},
			clones: []cloneTest{{
				vcs: hg,
				url: "/bitbucket.org/sqs/go-vcs-hgtest",
				ensureLocalFileExists: "foo",
			}},
			getFiles: []getFileTest{
				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/d047adf8d7ff0d3c589fe1d1cd72e1b8fb9512ea/foo", data: "Hello, foo"},
				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/default/foo", data: "Hello, foo"},
				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/default/bar", statusCode: http.StatusNotFound},
				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/barbranch/bar", data: "Hello, bar"},
				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/barbranch/foo", data: "Hello, foo"},

				{url: "/bitbucket.org/sqs/go-vcs-hgtest/v/doesntexist/foo", statusCode: http.StatusNotFound},
			},
		},
	}

	for i, test := range tests {
		if i != 1 {
			continue
		}
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

	for _, clone := range test.clones {
		testClone(t, clone, s.URL)
	}

	for _, getFile := range test.getFiles {
		testGetFile(t, getFile, s.URL)
	}
}

func testClone(t *testing.T, test cloneTest, serverURL string) {
	actionkey := "clone:" + test.url[1:]
	pre := actions[actionkey]

	// Make a temp dir for the client to clone the repo into.
	tmpdir, err := ioutil.TempDir("", "vcsserver-local")
	if err != nil {
		t.Fatal("TempDir failed:", err)
	}
	defer os.RemoveAll(tmpdir)

	localRepoDir := filepath.Join(tmpdir, "repo")
	_, err = test.vcs.Clone(serverURL+test.url, localRepoDir)
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

	if f := filepath.Join(localRepoDir, test.ensureLocalFileExists); !isFile(f) {
		t.Errorf("want file %s to exist", f)
	}

	var ok bool
	storedRepoDir := filepath.Join(StorageDir, test.vcs.ShortName()+"/"+test.url)
	switch test.vcs {
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
	if !ok {
		t.Errorf("no storedRepoDir contains a cloned repo (did the repo get cloned by the mapping handler?)")
	}

	testUpdate(t, test, serverURL, localRepoDir)

	if post := actions[actionkey]; post != pre+1 {
		t.Errorf("want 1 %s to have occurred during clone, got %d", actionkey, post-pre)
	}
}

func testUpdate(t *testing.T, test cloneTest, serverURL string, repodir string) {
	actionkey := "update:" + test.url[1:]
	pre := actions[actionkey]

	repo, err := test.vcs.Open(repodir)
	if err != nil {
		t.Fatal("Open failed:", err)
	}
	err = repo.Download()
	if err != nil {
		t.Fatal("Download failed:", err)
	}

	if post := actions[actionkey]; post != pre+1 {
		t.Errorf("want 1 %s to have occurred during clone, got %d", actionkey, post-pre)
	}
}

func testGetFile(t *testing.T, test getFileTest, serverURL string) {
	if test.statusCode == 0 {
		test.statusCode = http.StatusOK
	}

	data, statusCode := httpGET(t, serverURL+test.url)
	data = strings.TrimSpace(data)
	if statusCode != test.statusCode {
		t.Errorf("%s: want statusCode == %d, got %d", test.url, test.statusCode, statusCode)
	}
	return

	if data != test.data {
		t.Errorf("%s: want data == %q, got %q", test.url, test.data, data)
	}
}

// isFile returns true if path is an existing directory, and false otherwise.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func httpGET(t *testing.T, url string) (data string, status int) {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("httpGET %s: %s", url, err)
	}
	defer resp.Body.Close()
	return string(readAll(t, resp.Body)), resp.StatusCode
}

func readAll(t *testing.T, rdr io.Reader) []byte {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		t.Fatal("ReadAll", err)
	}
	return data
}
