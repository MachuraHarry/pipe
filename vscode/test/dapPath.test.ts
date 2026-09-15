import { describe, it, expect } from 'vitest';
import * as path from 'path';
import { resolveDapPath, dapBinaryName, type DapPathEnv } from '../src/dapPath';

function makeEnv(overrides: Partial<DapPathEnv> = {}): DapPathEnv {
	return {
		platform: 'linux',
		getConfigDapPath: () => '',
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

describe('resolveDapPath', () => {
	it('returns the configured dapPath first', () => {
		const env = makeEnv({ getConfigDapPath: () => '/custom/pipe-dap' });
		expect(resolveDapPath(env)).toBe('/custom/pipe-dap');
	});

	it('falls back to <workspace>/bin/pipe-dap when configured', () => {
		const ws = path.join('/ws', 'repo');
		const candidate = path.join(ws, 'bin', 'pipe-dap');
		const env = makeEnv({
			workspaceFolders: () => [ws],
			isFile: fileExists([candidate]),
		});
		expect(resolveDapPath(env)).toBe(candidate);
	});

	it('falls back to PATH when nothing is found in the workspace', () => {
		const env = makeEnv({ which: () => '/usr/local/bin/pipe-dap' });
		expect(resolveDapPath(env)).toBe('/usr/local/bin/pipe-dap');
	});

	it('returns undefined when nothing resolves', () => {
		expect(resolveDapPath(makeEnv())).toBeUndefined();
	});
});

describe('dapBinaryName', () => {
	it('uses the .exe suffix on Windows', () => {
		expect(dapBinaryName('win32')).toBe('pipe-dap.exe');
	});

	it('is a bare name elsewhere', () => {
		expect(dapBinaryName('linux')).toBe('pipe-dap');
		expect(dapBinaryName('darwin')).toBe('pipe-dap');
	});
});
