package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func SomeLogic(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second*5)
	log.Printf("hello from shazam logic.go")
	
	w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, "Hello, you've reached the Go server!")
}