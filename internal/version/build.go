package version

// Build is set by bin/pre_compile on Scalingo or via Dockerfile ARG on local builds.
// Default "dev" is used when building without a version (e.g. go run).
var Build = "dev"
