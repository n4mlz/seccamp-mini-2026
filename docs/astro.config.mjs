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
				{ label: '講義の概要', slug: 'overview' },
				{
					label: '事前学習',
					items: [
						{ label: '事前学習について', slug: 'prerequisites' },
						{ label: '1. Go の基本', slug: 'prerequisites/go' },
						{ label: '2. 開発環境', slug: 'prerequisites/environment' },
						{ label: '3. 応募課題の解説', slug: 'prerequisites/process-and-syscall' },
					],
				},
			],
		}),
	],
});
