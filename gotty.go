package main

import (
	"context"

	"github.com/sorenisanerd/gotty/backend/localcommand"
	"github.com/sorenisanerd/gotty/server"
	"github.com/sorenisanerd/gotty/utils"
)

func startGotty() {
	factory, err := localcommand.NewFactory("ssh", []string{"-o", "StrictHostKeyChecking=no", "-p", "2222", "127.0.0.1"}, &localcommand.Options{})
	if err != nil {
		panic(err)
	}
	appOptions := &server.Options{}
	appOptions.TitleVariables = map[string]interface{}{
		"command":  "ssh",
		"argv":     []string{""},
		"hostname": "",
	}
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
