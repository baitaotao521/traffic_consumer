package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/httpapi"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
)

func main() {
	dir, err := os.MkdirTemp("", "traffic-consumer-openapi-*")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(dir)

	paths := config.Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
	server := httpapi.New(appruntime.New(paths), ".")
	request := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	recorder := httptest.NewRecorder()
	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		fatal(fmt.Errorf("openapi schema status=%d body=%s", recorder.Code, recorder.Body.String()))
	}
	fmt.Print(recorder.Body.String())
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
