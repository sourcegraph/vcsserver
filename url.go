package vcsserver

import (
	"net/url"
	"strconv"
	"strings"
)

// ClonePath returns the HTTP request path on vcsserver that maps to cloneURL.
// Applications that use vcsserver to proxy repositories should construct clone
// URLs with the host URL of vcsserver and the path returned by this function.
func ClonePath(vcs string, cloneURL *url.URL) *url.URL {
	numPathComponents := strings.Count(cloneURL.Path, "/")
	return &url.URL{Path: "/" + strconv.Itoa(numPathComponents) + "/" + vcs + "/" + cloneURL.Scheme + "/" + cloneURL.Host + cloneURL.Path}
}

// FilePath returns the HTTP request path on vcsserver that maps to the
// specified file at revision. Applications that use vcsserver to proxy
// repositories should construct file URLs with the host URL of vcsserver and
// the path returned by this function.
func FilePath(vcs string, cloneURL *url.URL, revision, file string) *url.URL {
	return &url.URL{Path: ClonePath(vcs, cloneURL).Path + "/v/" + revision + "/" + file}
}

// BatchFilesURI returns the HTTP request URI on vcsserver that maps to a batch
// request of the specified files at revision. Applications that use vcsserver
// to proxy repositories should construct file URLs with the host URL of
// vcsserver and the URI returned by this function.
func BatchFilesURI(vcs string, cloneURL *url.URL, revision string, files []string) *url.URL {
	q := make(url.Values)
	q.Set("return", returnFirstExist)
	q["file"] = files
	return &url.URL{Path: ClonePath(vcs, cloneURL).Path + "/v-batch/" + revision, RawQuery: q.Encode()}
}
