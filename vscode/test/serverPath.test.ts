import { describe, it, expect } from 'vitest';
import * as path from 'path';
import {
	resolveServerPath,
	platformDir,
	lspBinaryName,
	type ServerPathEnv,
} from '../src/serverPath';

function makeEnv(overrides: Partial<ServerPathEnv> = {}): ServerPathEnv {
	return {
		platform: 'linux',
		arch: 'x64',
		getConfigLspPath: () => '',
		workspaceFolders: () => [],
		isFile: () => false,
		which: () => undefined,
		...overrides,
	};
}

function fileExists(files: string[]): (p: string) => boolean {
	const set = new Set(files.map((f) => path.normalize(f)));
	return (p: string) => set.has(path.normalize(p));
}

describe('resolveServerPath', () => {
	const ext = '/ext';

	it('returns the configured lspPath first', () => {
		const env = makeEnv({ getConfigLspPath: () => '/custom/pipe-lsp' });
		expect(resolveServerPath(ext, env)).toBe('/custom/pipe-lsp');
	});

	it('prefers the per-platform binary over the legacy layout', () => {
		const perPlatform = path.join(ext, 'bin', 'linux-amd64', 'pipe-lsp');
		const legacy = path.join(ext, 'bin', 'pipe-lsp');
		const env = makeEnv({ isFile: fileExists([perPlatform, legacy]) });
		expect(resolveServerPath(ext, env)).toBe(perPlatform);
	});

	it('falls back to the legacy layout when the per-platform binary is missing', () => {
		const perPlatform = path.join(ext, 'bin', 'linux-amd64', 'pipe-lsp');
		const legacy = path.join(ext, 'bin', 'pipe-lsp');
		const env = makeEnv({ isFile: fileExists([legacy]) });
		expect(env.isFile(perPlatform)).toBe(false);
		expect(resolveServerPath(ext, env)).toBe(legacy);
	});

	it('falls back to a workspace bin/pipe-lsp when the extension has no binary', () => {
		const ws = path.join('/ws', 'repo');
		const candidate = path.join(ws, 'bin', 'pipe-lsp');
		const env = makeEnv({
			workspaceFolders: () => [ws],
			isFile: fileExists([candidate]),
		});
		expect(resolveServerPath(ext, env)).toBe(candidate);
	});

	it('falls back to PATH when nothing is found in the extension or workspace', () => {
		const env = makeEnv({ which: () => '/usr/local/bin/pipe-lsp' });
		expect(resolveServerPath(ext, env)).toBe('/usr/local/bin/pipe-lsp');
	});

	it('returns undefined when nothing resolves', () => {
		expect(resolveServerPath(ext, makeEnv())).toBeUndefined();
	});
});

describe('platformDir', () => {
	it('maps supported platform/arch combos', () => {
		expect(platformDir('linux', 'x64')).toBe('linux-amd64');
		expect(platformDir('darwin', 'arm64')).toBe('darwin-arm64');
		expect(platformDir('win32', 'x64')).toBe('windows-amd64');
	});

	it('defaults unknown platforms and architectures', () => {
		expect(platformDir('freebsd', 'ppc64le')).toBe('linux-amd64');
	});
});

describe('lspBinaryName', () => {
	it('uses the .exe suffix on Windows', () => {
		expect(lspBinaryName('win32')).toBe('pipe-lsp.exe');
	});

	it('is a bare name elsewhere', () => {
		expect(lspBinaryName('linux')).toBe('pipe-lsp');
		expect(lspBinaryName('darwin')).toBe('pipe-lsp');
	});
});
