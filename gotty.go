package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/sorenisanerd/gotty/backend/localcommand"
	"github.com/sorenisanerd/gotty/server"
	"github.com/sorenisanerd/gotty/utils"
)

func init() {
	r, w := io.Pipe()
	log.SetOutput(w)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			for _, m := range []string{"Alternative URL"} {
				if strings.Contains(line, m) {
					goto CONTINUE
				}
			}
			fmt.Fprintln(os.Stderr, line)
		CONTINUE:
		}
	}()
}

func startGotty() {
	factory, err := localcommand.NewFactory("ssh", []string{"-o", "StrictHostKeyChecking=no", "-p", "2222", "127.0.0.1"}, &localcommand.Options{})
	if err != nil {
		panic(err)
	}
	appOptions := &server.Options{}
	_ = utils.ApplyDefaultValues(appOptions)
	appOptions.Address = "0.0.0.0"
	appOptions.Port = "8080"
	appOptions.PermitWrite = true
	srv, err := server.New(factory, appOptions)
	if err != nil {
		panic(err)
	}

	panic(srv.Run(context.Background()))
}
