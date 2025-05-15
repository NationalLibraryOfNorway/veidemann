package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	threadCount := 5

	flag.IntVar(&threadCount, "t", threadCount, "Set number of threads")
	flag.Parse()

	i := make(chan string, threadCount)

	for j := 0; j < threadCount; j++ {
		go func() {
			for s := range i {
				fmt.Print(rewrite(s))
			}
		}()
	}

	r := bufio.NewReader(os.Stdin)
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
		i <- l
	}
}

// rewrite matches lines for each requested URL with the
//
// see http://www.squid-cache.org/Doc/config/store_id_program/
func rewrite(s string) string {
	l := strings.TrimSpace(s)

	var (
		res       string
		channelId string
		url       string
		extras    string
	)

	parts := strings.SplitN(l, " ", 3)
	channelId = parts[0]
	url = parts[1]
	if len(parts) > 2 {
		extras = parts[2]
	}

	p := strings.SplitN(url, ":", 2)
	if extras == "-" || p[0] == "cache_object" {
		res = url
	} else {
		res = extras + url
	}

	return fmt.Sprintf("%s OK store-id=\"%s\"\n", channelId, res)
}
