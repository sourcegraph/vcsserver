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
