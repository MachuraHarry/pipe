import * as vscode from 'vscode';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
} from 'vscode-languageclient/node';

let client: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext): void {
	const config = vscode.workspace.getConfiguration('pipe');
	if (config.get<boolean>('lsp.enabled', true) === false) {
		return;
	}

	const serverPath = resolveServerPath(context);
	if (!serverPath) {
		vscode.window.showErrorMessage(
			'Pipe: pipe-lsp binary not found. Build it with `make lsp` or set the `pipe.lspPath` setting.'
		);
		return;
	}

		const outputChannel = vscode.window.createOutputChannel('Pipe Language Server');

	const serverOptions: ServerOptions = {
		run: { command: serverPath, args: [], transport: TransportKind.stdio },
		debug: { command: serverPath, args: [], transport: TransportKind.stdio },
	};

	const clientOptions: LanguageClientOptions = {
		documentSelector: [{ language: 'pipe', scheme: 'file' }],
		outputChannel,
	};

	client = new LanguageClient(
		'pipeLanguageServer',
		'Pipe Language Server',
		serverOptions,
		clientOptions
	);

	client.onDidChangeState((e) => {
		outputChannel.appendLine(`[pipe] state: ${e.newState}`);
	});

	outputChannel.appendLine(`[pipe] using server binary: ${serverPath}`);
	client.start().then(undefined, (err: unknown) => {
		const msg = err instanceof Error ? err.message : String(err);
		outputChannel.appendLine(`[pipe] failed to start language server: ${msg}`);
	});
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}

function resolveServerPath(context: vscode.ExtensionContext): string | undefined {
	// 1. Explicit configuration.
	const configured = vscode.workspace.getConfiguration('pipe').get<string>('lspPath', '');
	if (configured) {
		return configured;
	}

	// 2. Binary shipped inside the extension.
	const extBin = path.join(context.extensionPath, 'bin', 'pipe-lsp');
	if (isFile(extBin)) {
		return extBin;
	}

	// 3. <workspace>/bin/pipe-lsp (repo layout).
	for (const folder of vscode.workspace.workspaceFolders ?? []) {
		const candidate = path.join(folder.uri.fsPath, 'bin', 'pipe-lsp');
		if (isFile(candidate)) {
			return candidate;
		}
	}

	// 4. On PATH.
	const found = which('pipe-lsp');
	if (found) {
		return found;
	}

	return undefined;
}

function isFile(p: string): boolean {
	try {
		return fs.statSync(p).isFile();
	} catch {
		return false;
	}
}

function which(cmd: string): string | undefined {
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
