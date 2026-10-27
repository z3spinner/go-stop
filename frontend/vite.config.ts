import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vitest/config';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			strategy: ['custom-lang', 'preferredLanguage', 'baseLocale']
		}),
		sveltekit({
			preprocess: vitePreprocess(),
			compilerOptions: {
				runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
			},
			adapter: adapter({
				pages: '../web/build',
				assets: '../web/build',
				fallback: 'index.html',
				precompress: false,
				strict: false
			})
		}),
		svelteTesting()
	],
	server: {
		// Proxy /api to the Go backend. In Docker the target is the `app` service
		// (set VITE_API_PROXY_TARGET=http://app:8080); locally it defaults to localhost.
		proxy: { '/api': process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080' }
	},
	test: { environment: 'jsdom', globals: true, setupFiles: ['./vitest-setup.js'] }
});
