package vcsserver

import (
	"net/url"
	"testing"
)

func TestClonePath(t *testing.T) {
	tests := []struct {
		vcs           string
		cloneURL      string
		wantClonePath string
	}{
		{"git", "git://example.com/foo.git", "/1/git/git/example.com/foo.git"},
		{"git", "https://example.com/foo/bar.git", "/2/git/https/example.com/foo/bar.git"},
	}

	for _, test := range tests {
		cloneURL, err := url.Parse(test.cloneURL)
		if err != nil {
			t.Errorf("%s: url.Parse failed: %s", test.cloneURL, err)
			continue
		}
		clonePath := ClonePath(test.vcs, cloneURL)
		if test.wantClonePath != clonePath {
			t.Errorf("%s: want clonePath %s, got %s", test.cloneURL, test.wantClonePath, clonePath)
		}
	}
}

func TestFilePath(t *testing.T) {
	tests := []struct {
		vcs          string
		cloneURL     string
		revision     string
		file         string
		wantFilePath string
	}{
		{"git", "git://example.com/foo.git", "master", "foo.txt", "/1/git/git/example.com/foo.git/v/master/foo.txt"},
		{"git", "https://example.com/foo/bar.git", "1234abcdef", "my/file.txt", "/2/git/https/example.com/foo/bar.git/v/1234abcdef/my/file.txt"},
	}

	for _, test := range tests {
		cloneURL, err := url.Parse(test.cloneURL)
		if err != nil {
			t.Errorf("%s: url.Parse failed: %s", test.cloneURL, err)
			continue
		}
		filePath := FilePath(test.vcs, cloneURL, test.revision, test.file)
		if test.wantFilePath != filePath {
			t.Errorf("%s %s: want filePath %s, got %s", test.cloneURL, test.file, test.wantFilePath, filePath)
		}
	}
}
