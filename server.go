package main

import (
	"log"
	"net/http"
)

func main() {
	port := ":8000"
	dir := "./build"

	log.Printf("Starting server on http://localhost%s", port)
	log.Printf("Serving files from: %s", dir)

	err := http.ListenAndServe(port, http.FileServer(http.Dir(dir)))
	if err != nil {
		log.Fatal(err)
	}
}
