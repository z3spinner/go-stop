import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			strategy: ['custom-lang', 'preferredLanguage', 'baseLocale']
		}),
		sveltekit()
	],
	server: { proxy: { '/api': 'http://localhost:8080' } },
	test: { environment: 'jsdom', globals: true, setupFiles: ['./vitest-setup.js'] }
});
