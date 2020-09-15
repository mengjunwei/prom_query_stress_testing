package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	PHS *PrometheusHttpService
)

func init() {
	http.Handle("/metrics", promhttp.Handler())
}

type PrometheusHttpService struct {
	Service *http.Server
}

func NewPHS(addr string, handler http.Handler) *PrometheusHttpService {
	server := &http.Server{Addr: addr, Handler: handler}
	PHS = &PrometheusHttpService{Service: server}
	return PHS
}

func (h *PrometheusHttpService) Start() error {
	if h.Service != nil {
		return h.Service.ListenAndServe()
	}
	return nil
}

func (h *PrometheusHttpService) Stop() error {
	if h.Service != nil {
		return h.Service.Close()
	}
	return nil
}
