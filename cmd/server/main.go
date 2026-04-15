package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/timur/claude-code-openai-server/internal/proxy"
)

func main() {
	log.Println("Discovering Claude models...")
	reg := proxy.Discover([]string{"opus", "sonnet", "haiku"})
	if reg.Len() == 0 {
		log.Fatal("no Claude models discovered — ensure the claude CLI is on PATH and authenticated")
	}
	log.Printf("Discovered %d model(s)", reg.Len())

	h := &proxy.Handler{Registry: reg}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	http.HandleFunc("/v1/models", h.Models)
	http.HandleFunc("/v1/chat/completions", h.ChatCompletions)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
