package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "storeid: %v\n", err)
		os.Exit(1)
	}
}

func run(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)

	// Increase the default 64 KiB Scanner limit for unusually long URLs.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	writer := bufio.NewWriterSize(output, 64*1024)
	defer func() { _ = writer.Flush() }()

	for scanner.Scan() {
		if _, err := writer.WriteString(rewrite(scanner.Text())); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("flush response: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read request: %w", err)
	}

	return nil
}

func rewrite(input string) string {
	line := input
	if line == "" {
		return "BH message=empty-request\n"
	}

	url, extras, found := strings.Cut(line, " ")
	if found {
		extras = strings.TrimSpace(extras)
	} else {
		url = line
		extras = ""
	}

	if url == "" {
		return "BH message=missing-url\n"
	}

	if extras == "" || extras == "-" {
		return "ERR\n"
	}

	return fmt.Sprintf("OK store-id=v1|%s|%s\n", extras, url)
}
