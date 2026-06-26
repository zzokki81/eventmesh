// Package appenv defines the deployment profile a service runs under
// (development or production). The profile is the single switch from which
// environment-specific defaults are derived — for example whether to expose
// debug endpoints like pprof, or which log format to default to.
package appenv

// Environment names the deployment profile.
type Environment string

const (
	// Development is the local/dev profile: debug endpoints on, human-readable
	// logs by default.
	Development Environment = "development"

	// Production is the deployed profile: debug endpoints off, structured logs
	// by default.
	Production Environment = "production"
)

// IsDevelopment reports whether e is the development profile.
func (e Environment) IsDevelopment() bool { return e == Development }

// IsProduction reports whether e is the production profile.
func (e Environment) IsProduction() bool { return e == Production }
