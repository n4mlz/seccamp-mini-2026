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
					],
				},
				{
					label: '第2章 作って理解する',
					items: [
						{ label: '1. プロセスとシステムコール', slug: 'part2/process-and-syscall' },
						{ label: '2. ハンズオンの進め方', slug: 'part2/handson-guide' },
						{ label: '3. 環境構築', slug: 'part2/setup' },
						{ label: '4. ステップ 0: プロセスの起動', slug: 'part2/step-0-process' },
						{ label: '5. ステップ 1: PID namespace', slug: 'part2/step-1-pid-namespace' },
						{ label: '6. ステップ 2: mount namespace と rootfs', slug: 'part2/step-2-mount-namespace' },
						{ label: '7. ステップ 3: cgroup', slug: 'part2/step-3-cgroup' },
						{ label: '8. ステップ 4: user namespace (発展)', slug: 'part2/step-4-user-namespace' },
						{ label: '9. ステップ 5: network namespace (発展)', slug: 'part2/step-5-network-namespace' },
						{ label: '10. ステップ 6: capability と seccomp (発展)', slug: 'part2/step-6-capability-seccomp' },
						{ label: '11. トラブルシューティング', slug: 'part2/troubleshooting' },
					],
				},
			],
		}),
	],
});
