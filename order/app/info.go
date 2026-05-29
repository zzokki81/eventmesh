package app

import (
	"runtime"

	"github.com/zzokki81/eventmesh/order/transports/http/handlers"
)

var (
	Version    = "dev"
	CommitHash = "unknown"
	BuildDate  = "unknown"
)

const ServiceName = "order"

type Info struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

func CurrentInfo() Info {
	return Info{
		Service:   ServiceName,
		Version:   Version,
		Commit:    CommitHash,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}
}

func toHandlerInfo(i Info) handlers.InfoData {
	return handlers.InfoData{
		Service:   i.Service,
		Version:   i.Version,
		Commit:    i.Commit,
		BuildDate: i.BuildDate,
		GoVersion: i.GoVersion,
	}
}
