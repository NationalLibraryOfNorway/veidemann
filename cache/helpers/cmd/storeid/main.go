package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

func main() {
	threadCount := 5
	flag.IntVar(&threadCount, "t", threadCount, "Set number of workers")
	flag.Parse()

	in := make(chan string, threadCount*2)
	out := make(chan string, threadCount*2)

	// Single stdout writer to prevent interleaving/corruption.
	go func() {
		w := bufio.NewWriter(os.Stdout)
		for s := range out {
			if s == "" {
				s = "ERR\n"
			}
			_, _ = w.WriteString(s)
			_ = w.Flush()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(threadCount)
	for j := 0; j < threadCount; j++ {
		go func() {
			defer wg.Done()
			for s := range in {
				out <- rewriteSafe(s)
			}
		}()
	}

	r := bufio.NewReader(os.Stdin)
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			// Helpers should not panic; emit nothing and exit non-zero is also OK,
			// but simplest: break.
			break
		}
		in <- l
	}

	close(in)
	wg.Wait()
	close(out)
}

func rewriteSafe(s string) string {
	line := strings.TrimRight(s, "\r\n")
	if line == "" {
		return ""
	}

	// Parse: <channel-id> SP <url> [SP <extras...>]
	i := strings.IndexByte(line, ' ')
	if i < 0 {
		return fmt.Sprintf("%s ERR\n", line)
	}
	channelId := line[:i]

	rest := strings.TrimLeft(line[i+1:], " ")
	if rest == "" {
		return fmt.Sprintf("%s ERR\n", channelId)
	}
	j := strings.IndexByte(rest, ' ')
	url := rest
	extras := ""
	if j >= 0 {
		url = rest[:j]
		extras = strings.TrimLeft(rest[j+1:], " ")
	}

	if extras == "-" || extras == "" || strings.HasPrefix(url, "cache_object:") {
		return fmt.Sprintf("%s OK store-id=%s\n", channelId, url)
	}

	res := "v1|" + extras + "|" + url
	return fmt.Sprintf("%s OK store-id=%s\n", channelId, res)
}
