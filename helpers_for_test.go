package vcsserver

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

// isFile returns true if path is an existing directory, and false otherwise.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func httpGET(t *testing.T, url string) (data string, status int) {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("httpGET %s: %s", url, err)
	}
	defer resp.Body.Close()
	return string(readAll(t, resp.Body)), resp.StatusCode
}

func readAll(t *testing.T, rdr io.Reader) []byte {
	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		t.Fatal("ReadAll", err)
	}
	return data
}
