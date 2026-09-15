import * as path from 'path';

export interface CliPathEnv {
	platform: string;
	getConfigCliPath: () => string;
	workspaceFolders: () => string[];
	isFile: (p: string) => boolean;
	which: (cmd: string) => string | undefined;
}

export function cliBinaryName(platform: string): string {
	return platform === 'win32' ? 'pipe.exe' : 'pipe';
}

// resolveCliPath is deliberately separate from serverPath.ts's
// resolveServerPath: that function's contract (a per-platform binary
// bundled inside the extension, plus the pipe.lspPath setting) is specific
// to pipe-lsp, which the extension physically ships. The pipe CLI is never
// bundled, so only the workspace-layout and PATH lookups apply here.
export function resolveCliPath(env: CliPathEnv): string | undefined {
	// 1. Explicit configuration.
	const configured = env.getConfigCliPath();
	if (configured) {
		return configured;
	}

	// 2. <workspace>/bin/pipe (matches `make build`'s output layout).
	for (const folder of env.workspaceFolders()) {
		const candidate = path.join(folder, 'bin', cliBinaryName(env.platform));
		if (env.isFile(candidate)) {
			return candidate;
		}
	}

	// 3. On PATH.
	return env.which(cliBinaryName(env.platform));
}
