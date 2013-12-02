package vcsserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type blameTestGroup struct {
	handler *Handler
	blames  []blameTest
}

type blameTest struct {
	url        string
	statusCode int
	numHunks   int
	numCommits int
}

func TestBlameHandler(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "vcsserver")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	tests := []blameTestGroup{
		{
			handler: New([]string{"github.com"}),
			blames: []blameTest{
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/api/blame?v=d3dd4c84e9e429e28e05d53a04651bce084f0565", numHunks: 1, numCommits: 1},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/api/blame?v=master", numHunks: 2, numCommits: 2},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/api/blame?v=quxbranch", numHunks: 3, numCommits: 3},
				{url: "/2/git/git/github.com/sqs/vcsserver-gittest/api/blame?v=doesntexist", statusCode: http.StatusInternalServerError}, // TODO(sqs): should be http.StatusNotFound
			},
		},
	}

	for i, test := range tests {
		StorageDir = filepath.Join(tmpdir, strconv.Itoa(i))
		err = os.MkdirAll(StorageDir, 0755)
		if err != nil {
			t.Fatal("MkdirAll failed:", err)
		}
		groupTestBlame(t, test)
	}
}

func groupTestBlame(t *testing.T, test blameTestGroup) {
	mux := http.NewServeMux()
	mux.Handle("/", test.handler)
	s := httptest.NewServer(mux)
	defer s.Close()

	for _, blame := range test.blames {
		testBlame(t, blame, s.URL)
	}
}

func testBlame(t *testing.T, test blameTest, serverURL string) {
	if test.statusCode == 0 {
		test.statusCode = http.StatusOK
	}

	data, statusCode := httpGET(t, serverURL+test.url)
	if statusCode != test.statusCode {
		t.Errorf("%s: want statusCode == %d, got %d", test.url, test.statusCode, statusCode)
		return
	}
	if test.statusCode != 200 {
		return
	}

	var br BlameResponse
	err := json.Unmarshal([]byte(data), &br)
	if err != nil {
		t.Error("Unmarshal:", err)
		return
	}

	if test.numCommits != len(br.Commits) {
		t.Errorf("%s: want %d commits, got %d", test.url, test.numCommits, len(br.Commits))
	}
	if test.numHunks != len(br.Hunks) {
		t.Errorf("%s: want %d hunks, got %d", test.url, test.numHunks, len(br.Hunks))
	}
}
