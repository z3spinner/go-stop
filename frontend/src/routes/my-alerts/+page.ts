// SPDX-FileCopyrightText: 2026 Zeno Kerr
// SPDX-License-Identifier: AGPL-3.0-or-later

import { redirect } from '@sveltejs/kit';
export const load = () => { throw redirect(307, '/my-searches'); };
