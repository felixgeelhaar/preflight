// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://felixgeelhaar.github.io',
	base: '/preflight',
	integrations: [
		starlight({
			title: 'Preflight',
			description: 'Deterministic workstation compiler - compile declarative configuration into reproducible machine setups.',
			logo: {
				src: './src/assets/logo-mark.svg',
				replacesTitle: false,
			},
			favicon: '/favicon.svg',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/felixgeelhaar/preflight' },
			],
			editLink: {
				baseUrl: 'https://github.com/felixgeelhaar/preflight/edit/main/website/',
			},
			customCss: [
				'./src/styles/custom.css',
			],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'getting-started/introduction' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Quick Start', slug: 'getting-started/quickstart' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Configuration', slug: 'guides/configuration' },
						{ label: 'Layers & Targets', slug: 'guides/layers' },
						{ label: 'Providers', slug: 'guides/providers' },
						{ label: 'Plugins', slug: 'guides/plugins' },
						{ label: 'Plugin Security', slug: 'guides/security' },
						{ label: 'Dotfile Management', slug: 'guides/dotfiles' },
						{ label: 'Migration', slug: 'guides/migration' },
						{ label: 'Troubleshooting', slug: 'guides/troubleshooting' },
					],
				},
				{
					label: 'CLI Reference',
					items: [
						{ label: 'Commands', slug: 'cli/commands' },
						{ label: 'Flags & Options', slug: 'cli/flags' },
					],
				},
				{
					label: 'Architecture',
					items: [
						{ label: 'Overview', slug: 'architecture/overview' },
						{ label: 'Domains', slug: 'architecture/domains' },
						{ label: 'Design Principles', slug: 'architecture/principles' },
					],
				},
				{
					label: 'Development',
					items: [
						{ label: 'TDD Workflow', slug: 'development/tdd' },
						{ label: 'Contributing', slug: 'development/contributing' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Product Requirements', slug: 'reference/prd' },
						{ label: 'Vision', slug: 'reference/vision' },
						{ label: 'Changelog', slug: 'reference/changelog' },
					],
				},
			],
			head: [
				{
					tag: 'meta',
					attrs: {
						property: 'og:image',
						content: 'https://felixgeelhaar.github.io/preflight/og-image.svg',
					},
				},
				{
					tag: 'meta',
					attrs: {
						property: 'og:image:width',
						content: '1200',
					},
				},
				{
					tag: 'meta',
					attrs: {
						property: 'og:image:height',
						content: '630',
					},
				},
			],
		}),
	],
});
