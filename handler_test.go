package vcsserver

import (
	"github.com/sourcegraph/go-vcs"
	"reflect"
	"testing"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		hosts     []string
		path      string
		wantRoute *route
		wantErr   *httpError
	}{
		{
			hosts: []string{"example.com"},
			path:  "/1/git/git/example.com/myrepo/info/refs",
			wantRoute: &route{
				vcs:       vcs.Git,
				cloneURL:  "git://example.com/myrepo",
				uri:       "example.com/myrepo",
				action:    proxyAction,
				extraPath: "/info/refs",
			},
		},
		{
			hosts: []string{"example.com"},
			path:  "/1/git/git/example.com/myrepo",
			wantRoute: &route{
				vcs:       vcs.Git,
				cloneURL:  "git://example.com/myrepo",
				uri:       "example.com/myrepo",
				action:    proxyAction,
				extraPath: "",
			},
		},
		{
			hosts: []string{"example.com"},
			path:  "/1/git/git/example.com/myrepo/v/mybranch/mydir/myfile.txt",
			wantRoute: &route{
				vcs:       vcs.Git,
				cloneURL:  "git://example.com/myrepo",
				uri:       "example.com/myrepo",
				action:    singleFileAction,
				extraPath: "/v/mybranch/mydir/myfile.txt",
			},
		},
	}

	for _, test := range tests {
		route, err := router(test.hosts, test.path)
		if !reflect.DeepEqual(test.wantErr, err) {
			t.Errorf("%s: want err %v, got %v", test.path, test.wantErr, err)
			continue
		}
		if !reflect.DeepEqual(test.wantRoute, route) {
			t.Errorf("%s: want route %v, got %v", test.path, test.wantRoute, route)
		}
	}
}

func TestBisectBeforeNth(t *testing.T) {
	tests := []struct {
		s            string
		sep          string
		n            int
		want1, want2 string
	}{
		{s: "a/b", sep: "/", n: 1, want1: "a", want2: "/b"},
		{s: "a/b", sep: "/", n: 2, want1: "a/b", want2: ""},
		{s: "a/b/c/d", sep: "/", n: 2, want1: "a/b", want2: "/c/d"},
	}

	for _, test := range tests {
		got1, got2 := bisectBeforeNth(test.s, test.sep, test.n)
		if test.want1 != got1 {
			t.Errorf("(%s,%s,%d): want1 %q, got %q", test.s, test.sep, test.n, test.want1, got1)
		}
		if test.want2 != got2 {
			t.Errorf("(%s,%s,%d): want2 %q, got %q", test.s, test.sep, test.n, test.want2, got2)
		}
	}
}
