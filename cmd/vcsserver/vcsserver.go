package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/sourcegraph/go-vcs"
	"github.com/sourcegraph/vcsserver"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var bindAddr = flag.String("http", ":8080", "HTTP bind address")
var storageDir = flag.String("storage", "/tmp/vcsserver", "storage root dir for VCS repos")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vcsserver mirrors and serves VCS repositories.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tvcsserver [options] (request-path,host,repo-route,vcs-type,vcs-scheme)+\n\n")
		fmt.Fprintf(os.Stderr, "The options are:\n\n")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Example usage:\n\n")
		fmt.Fprintf(os.Stderr, "\tTo run a mirror mapping http://localhost:8080/github.com/user/repo.git to\n")
		fmt.Fprintf(os.Stderr, "\tgit repos at git://github.com/user/repo.git:\n")
		fmt.Fprintf(os.Stderr, "\t    $ vcsserver '/github.com/,github.com,^/([^/]+)/([^/])+,git,git'\n\n")
		fmt.Fprintf(os.Stderr, "\tTo run a mirror mapping http://localhost:8080/bitbucket.org/user/repo to\n")
		fmt.Fprintf(os.Stderr, "\tMercurial repos at https://bitbucket.org/user/repo:\n")
		fmt.Fprintf(os.Stderr, "\t    $ vcsserver '/bitbucket.org/,bitbucket.org,^/([^/]+)/([^/])+,hg,https'\n\n")
		fmt.Fprintf(os.Stderr, "\tTo run a mirror mapping http://localhost:8080/code.google.com/p/repo to\n")
		fmt.Fprintf(os.Stderr, "\tMercurial repos at https://code.google.com/p/repo:\n")
		fmt.Fprintf(os.Stderr, "\t    $ vcsserver '/code.google.com/,code.google.com,^/(p/[^/]+),hg,https'\n\n")
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

	for _, fieldstr := range flag.Args() {
		fields := strings.Split(fieldstr, ",")
		if want := 5; len(fields) != want {
			log.Fatalf("mapping must have %d comma-separated fields, got %d", want, len(fields))
		}

		var vcstype vcs.VCS
		var present bool
		if vcstype, present = vcs.VCSByName[fields[3]]; !present {
			log.Fatalf("unrecognized VCS type: %q", fields[1])
		}

		repo, err := regexp.Compile(fields[2])
		if err != nil {
			log.Fatalf("bad repo route regexp: %s", err)
		}

		m := vcsserver.Mapping{
			Host:   fields[1],
			VCS:    vcstype,
			Repo:   repo,
			Scheme: fields[4],
		}
		http.Handle(fields[0], handlers.CombinedLoggingHandler(os.Stderr, m))

		log.Printf("loaded mapping %+v", m)
	}

	fmt.Fprintf(os.Stderr, "starting server on %s\n", *bindAddr)
	err := http.ListenAndServe(*bindAddr, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %s", err)
	}
}
