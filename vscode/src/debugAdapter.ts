import * as vscode from 'vscode';
import { isFile, which } from './serverPath';
import { resolveDapPath, DapPathEnv } from './dapPath';

function resolveDap(): string | undefined {
	const env: DapPathEnv = {
		platform: process.platform,
		getConfigDapPath: () =>
			vscode.workspace.getConfiguration('pipe').get<string>('dapPath', ''),
		workspaceFolders: () =>
			(vscode.workspace.workspaceFolders ?? []).map((f) => f.uri.fsPath),
		isFile,
		which,
	};
	return resolveDapPath(env);
}

export class PipeDebugAdapterDescriptorFactory
	implements vscode.DebugAdapterDescriptorFactory
{
	createDebugAdapterDescriptor(
		_session: vscode.DebugSession,
		_executable: vscode.DebugAdapterExecutable | undefined
	): vscode.ProviderResult<vscode.DebugAdapterDescriptor> {
		const dapPath = resolveDap();
		if (!dapPath) {
			vscode.window.showErrorMessage(
				'Pipe: pipe-dap binary not found. Build it with `make dap` or set the `pipe.dapPath` setting.'
			);
			return undefined;
		}
		return new vscode.DebugAdapterExecutable(dapPath, []);
	}
}

export function registerDebugAdapter(context: vscode.ExtensionContext): void {
	context.subscriptions.push(
		vscode.debug.registerDebugAdapterDescriptorFactory(
			'pipe',
			new PipeDebugAdapterDescriptorFactory()
		)
	);
}
