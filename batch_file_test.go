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

type batchFileTestGroup struct {
	handler *Handler
	batches []batchFileTest
}

type batchFileTest struct {
	url        string
	statusCode int
	data       string
}

func TestBatchFileHandler(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	tests := []batchFileTestGroup{
		{
			handler: New([]string{"github.com"}),
			batches: []batchFileTest{
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v-batch/d3dd4c84e9e429e28e05d53a04651bce084f0565?file=doesntexist&file=foo&return=first-exist", data: "Hello, foo"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v-batch/master?file=doesntexist&file=foo&return=first-exist", data: "Hello, foo!!!"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v-batch/master?file=qux&return=first-exist", statusCode: http.StatusNotFound},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v-batch/quxbranch?file=doesntexist&file=qux&return=first-exist", data: "Hello, qux"},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v-batch/quxbranch?file=doesntexist&file=foo&return=first-exist", data: "Hello, foo!!!"},

				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/v/doesntexist?file=foo&return=first-exist", statusCode: http.StatusNotFound},
			},
		},
		{
			handler: New([]string{"bitbucket.org"}),
			batches: []batchFileTest{
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/d047adf8d7ff0d3c589fe1d1cd72e1b8fb9512ea?file=doesntexist&file=foo&return=first-exist", data: "Hello, foo"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/default?file=doesntexist&file=foo&return=first-exist", data: "Hello, foo"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/default?file=bar&return=first-exist", statusCode: http.StatusNotFound},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/barbranch?file=doesntexist&file=bar&return=first-exist", data: "Hello, bar"},
				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/barbranch?file=foo&file=bar&return=first-exist", data: "Hello, foo"},

				{url: "/2/hg/https/bitbucket.org/sqs/go-vcs-hgtest/v-batch/doesntexist?file=foo&return=first-exist", statusCode: http.StatusNotFound},
			},
		},
	}

	for i, test := range tests {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}
		groupTestBatchFile(t, test)
	}
}

func groupTestBatchFile(t *testing.T, test batchFileTestGroup) {
	mux := http.NewServeMux()
	mux.Handle("/", test.handler)
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, batch := range test.batches {
		testBatchFile(t, batch, s.URL)
	}
}

func testBatchFile(t *testing.T, test batchFileTest, serverURL string) {
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
