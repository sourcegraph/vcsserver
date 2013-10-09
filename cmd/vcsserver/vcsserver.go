package main

import (
	"flag"
	"fmt"
	"github.com/sourcegraph/vcsserver"
	"log"
	"net/http"
	"os"
)

var bindAddr = flag.String("http", ":8080", "HTTP bind address")
var storageDir = flag.String("storage", "/tmp/vcsserver", "storage root dir for VCS repos")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vcsserver mirrors and serves VCS repositories.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tvcsserver [options] (clone-host)+\n\n")
		fmt.Fprintf(os.Stderr, "For each clone-host specified, vcsserver provides an HTTP proxy for cloning\n")
		fmt.Fprintf(os.Stderr, "repositories on the host.\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Example usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tTo run a proxy for git repositories on GitHub:\n")
		fmt.Fprintf(os.Stderr, "\t    $ vcsserver github.com\n")
		fmt.Fprintf(os.Stderr, "\tTo clone a repository via vcsserver:\n")
		fmt.Fprintf(os.Stderr, "\t    $ HTTP_PROXY=http://localhost:8080 git clone http://github.com/user/repo.git\n")
		fmt.Fprintf(os.Stderr, "\tTo access a specific file (on the 'master' branch) via HTTP:\n")
		fmt.Fprintf(os.Stderr, "\t    $ curl http://localhost:8080/git/github.com/user/repo.git/v/master/file.txt\n")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "vcsserver reads the following environment variables:\n\n")
		fmt.Fprintf(os.Stderr, "\tGIT_HTTP_BACKEND   path to the `git-http-backend` executable\n")
		fmt.Fprintf(os.Stderr, "\tPYTHON27           path to the Python 2.7 interpreter\n")
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
	}

	log.SetPrefix("")
	log.SetFlags(0)

	vcsserver.StorageDir = *storageDir

	cloneHosts := flag.Args()
	http.Handle("/", &vcsserver.Handler{cloneHosts})

	fmt.Fprintf(os.Stderr, "starting server on %s\n", *bindAddr)
	err := http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
