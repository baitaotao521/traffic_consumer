package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/baitaotao521/traffic_consumer/internal/cli"
	"github.com/baitaotao521/traffic_consumer/internal/config"
	"github.com/baitaotao521/traffic_consumer/internal/httpapi"
	appruntime "github.com/baitaotao521/traffic_consumer/internal/runtime"
)

func main() {
	paths := config.ResolvePaths()
	args, _, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if args.Help {
		return
	}
	if args.IsCLIMode() {
		if err := cli.Run(args, paths); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	manager := appruntime.New(paths)
	manager.LaunchAutoStartConfigs()

	baseDir, err := os.Getwd()
	if err != nil {
		baseDir = filepath.Dir(os.Args[0])
	}
	server := httpapi.New(manager, baseDir)
	fmt.Println("启动 Web UI, 访问 http://127.0.0.1:5001")
	if err := http.ListenAndServe("0.0.0.0:5001", server.Router()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
