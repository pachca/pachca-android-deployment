package handler

import (
	"fmt"
	"net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	HandlePachcaHook(w, r, http.DefaultClient)
}

func HandlePachcaHook(w http.ResponseWriter, r *http.Request, client *http.Client) {
	w.WriteHeader(http.StatusNoContent)
	fmt.Fprintf(w, "<h1>Hello from Pachca!</h1>")
}
