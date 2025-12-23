package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: loghelper <stdout|stderr>")
		os.Exit(2)
	}

	var sink *os.File
	switch os.Args[1] {
	case "stdout":
		sink = os.Stdout
	case "stderr":
		sink = os.Stderr
	default:
		fmt.Fprintf(os.Stderr, "unsupported sink %q\n", os.Args[1])
		os.Exit(2)
	}

	r := bufio.NewReaderSize(os.Stdin, 512*1024)

	for {
		line, err := r.ReadBytes('\n')

		if len(line) > 0 {
			// Squid logfile_daemon protocol:
			// L<data>\n  → write <data> verbatim
			if line[0] == 'L' {
				_, _ = sink.Write(line[1:])
			}
			// All other commands (F, b*, R, T, O, r*) are ignored.
		}

		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "loghelper read error: %v\n", err)
			}
			return
		}
	}
}
