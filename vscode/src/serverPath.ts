import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

export interface ServerPathEnv {
	platform: string;
	arch: string;
	getConfigLspPath: () => string;
	workspaceFolders: () => string[];
	isFile: (p: string) => boolean;
	which: (cmd: string) => string | undefined;
}

export function resolveServerPath(
	extensionPath: string,
	env: ServerPathEnv
): string | undefined {
	// 1. Explicit configuration.
	const configured = env.getConfigLspPath();
	if (configured) {
		return configured;
	}

	// 2. Binary shipped inside the extension (per-platform bin/<os>-<arch>/pipe-lsp).
	const extBin = path.join(
		extensionPath,
		'bin',
		platformDir(env.platform, env.arch),
		lspBinaryName(env.platform)
	);
	if (env.isFile(extBin)) {
		return extBin;
	}

	// 3. Legacy single-binary layout.
	const legacyBin = path.join(extensionPath, 'bin', 'pipe-lsp');
	if (env.isFile(legacyBin)) {
		return legacyBin;
	}

	// 4. <workspace>/bin/pipe-lsp (repo layout).
	for (const folder of env.workspaceFolders()) {
		const candidate = path.join(folder, 'bin', 'pipe-lsp');
		if (env.isFile(candidate)) {
			return candidate;
		}
	}

	// 5. On PATH.
	const found = env.which('pipe-lsp');
	if (found) {
		return found;
	}

	return undefined;
}

export function platformDir(platform: string, arch: string): string {
	const osMap: Record<string, string> = { linux: 'linux', darwin: 'darwin', win32: 'windows' };
	const archMap: Record<string, string> = { x64: 'amd64', arm64: 'arm64' };
	const os = osMap[platform] ?? 'linux';
	const a = archMap[arch] ?? 'amd64';
	return `${os}-${a}`;
}

export function lspBinaryName(platform: string): string {
	return platform === 'win32' ? 'pipe-lsp.exe' : 'pipe-lsp';
}

export function isFile(p: string): boolean {
	try {
		return fs.statSync(p).isFile();
	} catch {
		return false;
	}
}

export function which(cmd: string): string | undefined {
	try {
		const found = execSync(
			process.platform === 'win32' ? `where ${cmd}` : `command -v ${cmd}`,
			{ encoding: 'utf8' }
		)
			.trim()
			.split(/\r?\n/)[0];
		return found || undefined;
	} catch {
		return undefined;
	}
}
