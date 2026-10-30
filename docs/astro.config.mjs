// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import mermaid from 'astro-mermaid';

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
		mermaid({ autoTheme: true }),
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
				{
					label: '第1章 コンテナを使う側から理解する',
					items: [
						{ label: '1. コンテナとは', slug: 'part1/what-is-container' },
						{ label: '2. コンテナを使ってみる', slug: 'part1/try-container' },
						{ label: '3. なぜコンテナは広く使われているのか', slug: 'part1/why-used' },
						{ label: '4. コンテナはどのように使われているか、今こそ学ぶ意義', slug: 'part1/usecases-and-why-now' },
						{ label: '5. コンテナと VM の違い', slug: 'part1/vs-vm' },
					],
				},
			],
		}),
	],
});
