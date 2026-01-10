package api

import "github.com/vango-go/vango"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func HealthGET(ctx vango.Ctx) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Version: "0.1.0",
	}, nil
}
