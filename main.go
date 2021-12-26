package main

import (
	_ "embed"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

func init() {
	// Force truecolor in all terminal rendering, means we'll be trying to
	// render color in terminals that don't support it leading to strange rainbow rendering artifacts.
	//
	// Feature not a bug?
	os.Setenv("COLORTERM", "truecolor")
	rand.Seed(time.Now().UnixNano())
}

func main() {
	go func() { panic(http.ListenAndServe(":6060", nil)) }()
	Float{}.Start()
}
