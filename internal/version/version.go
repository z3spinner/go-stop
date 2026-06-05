// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

// Get returns the build version for cache-busting static assets.
// Build is set at compile time by bin/pre_compile (Scalingo) or
// the Dockerfile ARG (local dev). Falls back to "dev".
func Get() string {
	if Build != "" && Build != "dev" {
		return Build
	}
	return "dev"
}
