package vcsserver

import (
	"net/url"
	"strconv"
	"strings"
)

// ClonePath returns the HTTP request path on vcsserver that maps to cloneURL.
// Applications that use vcsserver to proxy repositories should construct clone
// URLs with the host URL of vcsserver and the path returned by this function.
func ClonePath(vcs string, cloneURL *url.URL) string {
	numPathComponents := strings.Count(cloneURL.Path, "/")
	return "/" + strconv.Itoa(numPathComponents) + "/" + vcs + "/" + cloneURL.Scheme + "/" + cloneURL.Host + cloneURL.Path
}
