package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		fmt.Printf("%s [%s]%s %s\n", time.Now().String(), r.Method, r.URL, bytes.NewBuffer(body).String())
	})
	log.Fatal(http.ListenAndServe("0.0.0.0:5001", nil))
}
