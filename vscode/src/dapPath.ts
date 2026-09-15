import * as path from 'path';

export interface DapPathEnv {
	platform: string;
	getConfigDapPath: () => string;
	workspaceFolders: () => string[];
	isFile: (p: string) => boolean;
	which: (cmd: string) => string | undefined;
}

export function dapBinaryName(platform: string): string {
	return platform === 'win32' ? 'pipe-dap.exe' : 'pipe-dap';
}

// resolveDapPath mirrors cliPath.ts's resolveCliPath rather than
// serverPath.ts's resolveServerPath: like the pipe CLI, pipe-dap is never
// bundled inside the extension, so only the pipe.dapPath setting, the
// <workspace>/bin layout, and PATH apply -- no per-platform extension-bin
// lookup.
export function resolveDapPath(env: DapPathEnv): string | undefined {
	// 1. Explicit configuration.
	const configured = env.getConfigDapPath();
	if (configured) {
		return configured;
	}

	// 2. <workspace>/bin/pipe-dap (matches `make dap`'s output layout).
	for (const folder of env.workspaceFolders()) {
		const candidate = path.join(folder, 'bin', dapBinaryName(env.platform));
		if (env.isFile(candidate)) {
			return candidate;
		}
	}

	// 3. On PATH.
	return env.which(dapBinaryName(env.platform));
}
