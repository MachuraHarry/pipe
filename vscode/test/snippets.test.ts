import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';

interface Snippet {
	prefix: string;
	body: string | string[];
	description?: string;
}

describe('snippets/pipe.json', () => {
	const raw = fs.readFileSync(path.join(__dirname, '..', 'snippets', 'pipe.json'), 'utf8');
	const snippets: Record<string, Snippet> = JSON.parse(raw);

	it('is non-empty', () => {
		expect(Object.keys(snippets).length).toBeGreaterThan(0);
	});

	it('every entry has a non-empty prefix, body, and description', () => {
		for (const [name, snippet] of Object.entries(snippets)) {
			expect(snippet.prefix, `${name}: prefix`).toBeTruthy();
			expect(snippet.description, `${name}: description`).toBeTruthy();
			const bodyLines = Array.isArray(snippet.body) ? snippet.body : [snippet.body];
			expect(bodyLines.length, `${name}: body`).toBeGreaterThan(0);
			expect(bodyLines.join('\n').trim(), `${name}: body`).not.toBe('');
		}
	});
});
