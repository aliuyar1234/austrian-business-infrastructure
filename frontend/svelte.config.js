import adapter from '@sveltejs/adapter-auto';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter(),
		alias: {
			'@/*': './src/*',
			'$lib': './src/lib',
			'$lib/*': './src/lib/*',
			'@components': './src/lib/components',
			'@components/*': './src/lib/components/*',
			'@stores': './src/lib/stores',
			'@stores/*': './src/lib/stores/*',
			'@api': './src/lib/api',
			'@api/*': './src/lib/api/*'
		}
	}
};

export default config;
