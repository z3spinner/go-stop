package version

import "os"

// Get returns the build version for cache-busting static assets.
// Scalingo sets SOURCE_VERSION to the git SHA on every deploy.
// Falls back to "dev" for local development.
func Get() string {
	if v := os.Getenv("SOURCE_VERSION"); v != "" {
		if len(v) > 8 {
			return v[:8]
		}
		return v
	}
	return "dev"
}
