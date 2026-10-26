// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

const [githubOwner, repositoryName] = process.env.GITHUB_REPOSITORY?.split('/') ?? [];
const site = githubOwner ? `https://${githubOwner}.github.io` : undefined;
const isProjectPage = Boolean(
	githubOwner && repositoryName && repositoryName !== `${githubOwner}.github.io`,
);

// https://astro.build/config
export default defineConfig({
	site,
	base: isProjectPage ? `/${repositoryName}` : '/',
	integrations: [
		starlight({
			title: '作って学ぶLinuxコンテナの仕組みとセキュリティ',
			sidebar: [
				{
					label: 'Guides',
					items: [
						// Each item here is one entry in the navigation menu.
						{ label: 'Example Guide', slug: 'guides/example' },
					],
				},
				{
					label: 'Reference',
					items: [{ autogenerate: { directory: 'reference' } }],
				},
			],
		}),
	],
});
