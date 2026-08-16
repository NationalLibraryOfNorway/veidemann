package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

func main() {
	certFile := flag.String("cert-file", "", "path for the generated PEM certificate")
	keyFile := flag.String("key-file", "", "path for the generated PEM private key")
	flag.Parse()

	if err := mitmcert.Write(*certFile, *keyFile, time.Now()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
