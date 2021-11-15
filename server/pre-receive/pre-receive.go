package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strings"
)

func main() {
	defer func() {
		err := recover()
		if err != nil {
			message := fmt.Sprintf("GL-HOOK-ERR: %+v, stack: %s", err, string(debug.Stack()))
			fmt.Printf("GL-HOOK-ERR: %s", message)
			ioutil.WriteFile("/tmp/pre-receive-panic.log", []byte(message), 0755)
			os.Exit(1)
		}
	}()

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
		hook.Info(ColorGreenBold, "scanner err: %s", err)
		os.Exit(0)
	}
}
