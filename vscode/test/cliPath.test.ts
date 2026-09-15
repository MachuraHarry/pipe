import { describe, it, expect } from 'vitest';
import * as path from 'path';
import { resolveCliPath, cliBinaryName, type CliPathEnv } from '../src/cliPath';

function makeEnv(overrides: Partial<CliPathEnv> = {}): CliPathEnv {
	return {
		platform: 'linux',
		getConfigCliPath: () => '',
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

describe('resolveCliPath', () => {
	it('returns the configured cliPath first', () => {
		const env = makeEnv({ getConfigCliPath: () => '/custom/pipe' });
		expect(resolveCliPath(env)).toBe('/custom/pipe');
	});

	it('falls back to <workspace>/bin/pipe when configured', () => {
		const ws = path.join('/ws', 'repo');
		const candidate = path.join(ws, 'bin', 'pipe');
		const env = makeEnv({
			workspaceFolders: () => [ws],
			isFile: fileExists([candidate]),
		});
		expect(resolveCliPath(env)).toBe(candidate);
	});

	it('falls back to PATH when nothing is found in the workspace', () => {
		const env = makeEnv({ which: () => '/usr/local/bin/pipe' });
		expect(resolveCliPath(env)).toBe('/usr/local/bin/pipe');
	});

	it('returns undefined when nothing resolves', () => {
		expect(resolveCliPath(makeEnv())).toBeUndefined();
	});
});

describe('cliBinaryName', () => {
	it('uses the .exe suffix on Windows', () => {
		expect(cliBinaryName('win32')).toBe('pipe.exe');
	});

	it('is a bare name elsewhere', () => {
		expect(cliBinaryName('linux')).toBe('pipe');
		expect(cliBinaryName('darwin')).toBe('pipe');
	});
});
