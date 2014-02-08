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

// FilePath returns the HTTP request path on vcsserver that maps to the
// specified file at revision. Applications that use vcsserver to proxy
// repositories should construct file URLs with the host URL of vcsserver and
// the path returned by this function.
func FilePath(vcs string, cloneURL *url.URL, revision, file string) string {
	return ClonePath(vcs, cloneURL) + "/v/" + revision + "/" + file
}

// BatchFilesPath returns the HTTP request path on vcsserver that maps to a
// batch request of the specified files at revision. Applications that use
// vcsserver to proxy repositories should construct file URLs with the host URL
// of vcsserver and the path returned by this function.
func BatchFilesPath(vcs string, cloneURL *url.URL, revision string, files []string) string {
	q := make(url.Values)
	q.Set("return", returnFirstExist)
	q["file"] = files
	return ClonePath(vcs, cloneURL) + "/v-batch/" + revision + "?" + q.Encode()
}
