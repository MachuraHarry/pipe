import * as vscode from 'vscode';
import { isFile, which } from './serverPath';
import { resolveCliPath, CliPathEnv } from './cliPath';

const TERMINAL_NAME = 'Pipe';

function getOrCreateTerminal(): vscode.Terminal {
	const existing = vscode.window.terminals.find(
		(t) => t.name === TERMINAL_NAME && t.exitStatus === undefined
	);
	return existing ?? vscode.window.createTerminal(TERMINAL_NAME);
}

function resolveCli(): string | undefined {
	const env: CliPathEnv = {
		platform: process.platform,
		getConfigCliPath: () =>
			vscode.workspace.getConfiguration('pipe').get<string>('cliPath', ''),
		workspaceFolders: () =>
			(vscode.workspace.workspaceFolders ?? []).map((f) => f.uri.fsPath),
		isFile,
		which,
	};
	return resolveCliPath(env);
}

function notFoundError(): void {
	vscode.window.showErrorMessage(
		'Pipe: pipe binary not found. Build it with `make build` or set the `pipe.cliPath` setting.'
	);
}

function quote(p: string): string {
	return /\s/.test(p) ? `"${p}"` : p;
}

async function runFile(vm: boolean): Promise<void> {
	const editor = vscode.window.activeTextEditor;
	if (!editor || editor.document.languageId !== 'pipe') {
		vscode.window.showErrorMessage('Pipe: open a .pipe file to run.');
		return;
	}
	await editor.document.save();
	const cli = resolveCli();
	if (!cli) {
		notFoundError();
		return;
	}
	const terminal = getOrCreateTerminal();
	terminal.show();
	terminal.sendText(`${quote(cli)} ${vm ? '-vm ' : ''}${quote(editor.document.fileName)}`);
}

function openRepl(): void {
	const cli = resolveCli();
	if (!cli) {
		notFoundError();
		return;
	}
	const terminal = getOrCreateTerminal();
	terminal.show();
	terminal.sendText(quote(cli));
}

export function registerCommands(context: vscode.ExtensionContext): void {
	context.subscriptions.push(
		vscode.commands.registerCommand('pipe.runFile', () => runFile(false)),
		vscode.commands.registerCommand('pipe.runFileVM', () => runFile(true)),
		vscode.commands.registerCommand('pipe.openRepl', () => openRepl())
	);
}
