package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	hook := &Hook{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		arr := strings.Split(strings.TrimSpace(scanner.Text()), " ")
		oldRev := arr[0]
		newRev := arr[1]
		ref := arr[2]
		code := hook.Run(oldRev, newRev, ref)
		if code > 0 {
			os.Exit(code)
		}
		_, err := hook.CommitLog()
		if err != nil {
			hook.Info(ColorRedBold, "commit log err: %s", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
