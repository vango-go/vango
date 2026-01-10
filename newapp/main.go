package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vango-go/vango"
	"newapp/app/routes"
)

func main() {
	app := vango.New(vango.Config{
		Session: vango.SessionConfig{
			ResumeWindow: 30 * time.Second,
		},
		Static: vango.StaticConfig{
			Dir:    "public",
			Prefix: "/",
		},
		DevMode: os.Getenv("ENVIRONMENT") != "production",
	})

	routes.Register(app)

	mux := http.NewServeMux()
	mux.Handle("/", app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
