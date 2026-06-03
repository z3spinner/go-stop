import { defineConfig } from 'orval';

export default defineConfig({
	gostop: {
		input: { target: '../docs/swagger.json' },
		output: {
			target: './src/lib/api/generated/go-stop-api.ts',
			mode: 'single',
			client: 'fetch',
			override: {
				mutator: { path: './src/lib/api/fetchMutator.ts', name: 'customFetch' }
			}
		}
	}
});
