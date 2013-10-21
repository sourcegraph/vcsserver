package vcsserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type fileTestGroup struct {
	handler *Handler
	files   []fileTest
}

type fileTest struct {
	url        string
	statusCode int
	data       string
}

func TestFileHandler(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	tests := []fileTestGroup{
		{
			handler: New([]string{"github.com"}),
			files: []fileTest{
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/d3dd4c84e9e429e28e05d53a04651bce084f0565/foo", data: "Hello, foo"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/master/foo", data: "Hello, foo!!!"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/master/qux", statusCode: http.StatusNotFound},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/quxbranch/qux", data: "Hello, qux"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/quxbranch/foo", data: "Hello, foo!!!"},

				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/doesntexist/foo", statusCode: http.StatusNotFound},
			},
		},
		{
			handler: New([]string{"bitbucket.org"}),
			files: []fileTest{
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/d047adf8d7ff0d3c589fe1d1cd72e1b8fb9512ea/foo", data: "Hello, foo"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/default/foo", data: "Hello, foo"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/default/bar", statusCode: http.StatusNotFound},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/barbranch/bar", data: "Hello, bar"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/barbranch/foo", data: "Hello, foo"},

				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v/doesntexist/foo", statusCode: http.StatusNotFound},
			},
		},
	}

	for i, test := range tests {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}
		groupTestFile(t, test)
	}
}

func groupTestFile(t *testing.T, test fileTestGroup) {
	mux := http.NewServeMux()
	mux.Handle("/", test.handler)
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, file := range test.files {
		testFile(t, file, s.URL)
	}
}

func testFile(t *testing.T, test fileTest, serverURL string) {
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
