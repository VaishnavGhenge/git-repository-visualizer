package main

import (
	"log"
	"net/http"

	"git-repository-visualizer/internal/config"
	internalHttp "git-repository-visualizer/internal/http"
)

func main() {
	cfg := config.Load()

	h := internalHttp.NewHandler()

	log.Printf("Server starting on port %s...", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h); err != nil {
		log.Fatal(err)
	}
}
