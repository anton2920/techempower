package main

import (
	"log"
	"net/http"
)

var HelloWorld = []byte("Hello, world!\n")

func main() {
	http.HandleFunc("/plaintext", func(w http.ResponseWriter, r *http.Request) {
		w.Write(HelloWorld)
	})
	log.Printf("Listening on 0.0.0.0:7073...")
	log.Fatal(http.ListenAndServe("0.0.0.0:7073", nil))
}
