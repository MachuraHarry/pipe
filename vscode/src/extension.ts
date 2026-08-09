import * as vscode from 'vscode';
import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind,
} from 'vscode-languageclient/node';
import { isFile, resolveServerPath, which } from './serverPath';

let client: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext): void {
	const config = vscode.workspace.getConfiguration('pipe');
	if (config.get<boolean>('lsp.enabled', true) === false) {
		return;
	}

	const serverPath = resolveServerPath(context.extensionPath, {
		platform: process.platform,
		arch: process.arch,
		getConfigLspPath: () =>
			vscode.workspace.getConfiguration('pipe').get<string>('lspPath', ''),
		workspaceFolders: () =>
			(vscode.workspace.workspaceFolders ?? []).map((f) => f.uri.fsPath),
		isFile,
		which,
	});
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
