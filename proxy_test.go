package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type proxyTestGroup struct {
	handler *Handler
	proxies []proxyTest
}

type proxyTest struct {
	vcs                   vcs.VCS
	url                   string
	uri                   string
	cloneURL              string
	ensureLocalFileExists string
}

func TestProxy(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	hg, git := vcs.VCSByName["hg"], vcs.VCSByName["git"]

	testGroups := []proxyTestGroup{
		{
			handler: New([]string{"github.com"}),
			proxies: []proxyTest{{
				vcs:                   git,
				uri:                   "github.com/sqs/vcsserver-gittest.git",
				cloneURL:              "git://github.com/sqs/vcsserver-gittest.git",
				ensureLocalFileExists: "foo",
			}},
		},
		{
			handler: New([]string{"bitbucket.org"}),
			proxies: []proxyTest{{
				vcs:                   hg,
				uri:                   "bitbucket.org/sqs/go-vcs-hgtest",
				cloneURL:              "https://bitbucket.org/sqs/go-vcs-hgtest",
				ensureLocalFileExists: "foo",
			}},
		},
	}

	for i, test := range testGroups {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}
		groupTestProxy(t, test)
	}
}

func groupTestProxy(t *testing.T, test proxyTestGroup) {
	mux := http.NewServeMux()
	mux.Handle("/", test.handler)
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, proxy := range test.proxies {
		testProxy(t, proxy, s.URL)
	}
}

func testProxy(t *testing.T, test proxyTest, serverURL string) {
	actionkey := "clone:" + test.cloneURL
	pre := actions[actionkey]

	// Make a temp dir for the client to clone the repo into.
	tmpdir, err := ioutil.TempDir("", "vcsserver-local")
	if err != nil {
		t.Fatal("TempDir failed:", err)
	}
	defer os.RemoveAll(tmpdir)

	cloneURL, _ := url.Parse(test.cloneURL)
	clonePath := ClonePath(test.vcs.ShortName(), cloneURL)

	localRepoDir := filepath.Join(tmpdir, "repo")
	_, err = test.vcs.Clone(serverURL+clonePath, localRepoDir)
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
	storedRepoDir := filepath.Join(StorageDir, test.vcs.ShortName()+"/"+test.uri)
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
		t.Errorf("want 1 %s to have occurred during proxy, got %d", actionkey, post-pre)
	}
}

func testUpdate(t *testing.T, test proxyTest, serverURL string, repodir string) {
	actionkey := "update:" + test.cloneURL
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
		t.Errorf("want 1 %s to have occurred during proxy, got %d", actionkey, post-pre)
	}
}
