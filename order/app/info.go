package app

import (
	"runtime"

	"github.com/zzokki81/eventmesh/pkg/httpserver/handler"
)

var (
	Version    = "dev"
	CommitHash = "unknown"
	BuildDate  = "unknown"
)

const ServiceName = "order"

func CurrentInfo() handler.InfoData {
	return handler.InfoData{
		Service:   ServiceName,
		Version:   Version,
		Commit:    CommitHash,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}
