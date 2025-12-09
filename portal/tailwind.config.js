/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				primary: 'var(--primary-color, #3B82F6)',
				secondary: 'var(--secondary-color, #64748B)',
				accent: 'var(--accent-color, #8B5CF6)'
			}
		}
	},
	plugins: []
};
