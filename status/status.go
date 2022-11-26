package status

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type Data struct {
	Uptime  string `json:"uptime"`
	Version string `json:"version"`
}

type Status struct {
	Host      string
	Port      string
	StartTime time.Time
	Data      Data
}

func New(host, port, version string, startTime time.Time) *Status {
	data := Data{Version: version}
	return &Status{
		Host:      host,
		Port:      port,
		StartTime: startTime,
		Data:      data,
	}
}

func (s *Status) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	s.Data.Uptime = fmt.Sprintf("%v", time.Since(s.StartTime).Seconds())
	if err := json.NewEncoder(w).Encode(s.Data); err != nil {
		w.WriteHeader(500)
		if _, err := w.Write([]byte(fmt.Sprintf("internal error: %v", err))); err != nil {
			log.WithError(err).Warn("failed to write error response")
		}
	}
}

func (s *Status) Serve() {
	mux := http.NewServeMux()
	mux.Handle("/status", s)
	log.Debug("Starting status server on: ", net.JoinHostPort(s.Host, s.Port))
	go func() {
		err := http.ListenAndServe(net.JoinHostPort(s.Host, s.Port), mux)
		if err != nil {
			log.Fatalln(err)
		}
	}()
}
