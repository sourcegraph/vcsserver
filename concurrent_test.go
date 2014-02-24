package vcsserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sourcegraph/go-vcs"
)

func TestConcurrentUpdate(t *testing.T) {
	// Can also test with:
	//	 tests := []string{"http://bitbucket.org/sqs/go-vcs-gittest.git"}
	tests := []string{"git://github.com/sqs/vcsserver-gittest.git"}
	for _, test := range tests {
		testConcurrentUpdate(t, test)
	}
}

func testConcurrentUpdate(t *testing.T, cloneURL string) {
	mux := http.NewServeMux()
	mux.Handle("/", New([]string{"github.com", "bitbucket.org"}))
	s := httptest.NewServer(mux)
	defer s.Close()

	vcs := vcs.VCSByName["git"]
	u, _ := url.Parse(cloneURL)
	clonePath := ClonePath(vcs.ShortName(), u)

	tmpdir, err := ioutil.TempDir("", "vcsserver-concurrent-test")
	if err != nil {
		t.Fatalf("TempDir:", err)
	}
	defer os.RemoveAll(tmpdir)

	var cloneToTemp = func(subdir string) error {
		_, err = vcs.Clone(s.URL+clonePath.String(), filepath.Join(tmpdir, subdir))
		return err
	}

	N := 25
	d := time.Second * 5
	done := make(chan error, N)
	for i := 0; i < N; i++ {
		go func(i int) {
			err := cloneToTemp("c" + strconv.Itoa(i))
			done <- err
		}(i)
	}
	timer := time.NewTimer(d)
	for i := 0; i < N; i++ {
		select {
		case err := <-done:
			if err != nil {
				timer.Stop()
				t.Errorf("Clone error (#%d): %s", i, err)
				return
			}
		case <-timer.C:
			t.Fatalf("Timeout: %s (%d finished)", d, i)
			return
		}
	}
}
