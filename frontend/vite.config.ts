import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	resolve: {
		alias: {
			'@': '/src',
			'@lib': '/src/lib',
			'@components': '/src/lib/components',
			'@stores': '/src/lib/stores',
			'@api': '/src/lib/api'
		}
	}
});
