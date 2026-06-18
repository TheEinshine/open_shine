package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Shine's Service v3 is running")
	})

	fmt.Println("Listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}