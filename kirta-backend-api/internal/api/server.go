package api

import (
	"kirta-backend-api/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewServer(g *gin.Engine, cfg config.HTTPConfig) *http.Server {
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           g,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}
