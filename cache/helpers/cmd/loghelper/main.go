package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	logger := log.New(os.Stderr, "[LogHelper] ", log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)

	r := bufio.NewReader(os.Stdin)
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Println(err)
		}
		l = strings.Trim(l, " \t\n\r")
		if strings.HasPrefix(l, "L") {
			l = l[1:]
			_, _ = fmt.Fprintln(os.Stderr, l)
		}
	}
}
