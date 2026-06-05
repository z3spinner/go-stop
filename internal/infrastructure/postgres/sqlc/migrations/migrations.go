// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

package migrations

import "embed"

//go:embed *.up.sql *.down.sql
var FS embed.FS
