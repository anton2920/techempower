package main

import (
	"log"

	"github.com/valyala/fasthttp"
)

var HelloWorld = []byte("Hello, world!\n")

func PlaintextHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Path()) == "/plaintext" {
		ctx.Write(HelloWorld)
	}
}

func main() {
	log.Printf("Listening on 0.0.0.0:7073...")
	log.Fatal(fasthttp.ListenAndServe("0.0.0.0:7073", PlaintextHandler))
}
