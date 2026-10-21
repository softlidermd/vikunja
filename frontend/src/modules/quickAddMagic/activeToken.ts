import type {Prefixes} from './types'

export type TokenKind = 'label' | 'assignee' | 'project'

export interface ActiveToken {
	kind: TokenKind
	prefix: string
	start: number
	end: number
	query: string
}

const KINDS: TokenKind[] = ['label', 'assignee', 'project']

/**
 * Finds the label, assignee or project token the caret is in, by the rules of prefixParser:
 * the prefix starts the text or follows whitespace, and the value runs to the next whitespace
 * or, when it starts with a quote, to the matching quote (or the end of the line).
 */
export function getActiveToken(text: string, caret: number, prefixes: Prefixes | undefined): ActiveToken | null {
	if (!prefixes) {
		return null
	}

	for (let start = caret - 1; start >= 0 && text[start] !== '\n'; start--) {
		const kind = KINDS.find(k => prefixes[k] === text[start])
		if (!kind || (start > 0 && !/\s/.test(text[start - 1]))) {
			continue
		}

		const typed = text.slice(start + 1, caret)
		const quote = typed[0]
		if (quote === '"' || quote === '\'') {
			if (typed.includes(quote, 1)) {
				return null
			}
			const lineEnd = text.indexOf('\n', start)
			const close = text.indexOf(quote, start + 2)
			const end = close !== -1 && (lineEnd === -1 || close < lineEnd)
				? close + 1
				: (lineEnd === -1 ? text.length : lineEnd)
			return {kind, prefix: text[start], start, end, query: typed.slice(1)}
		}

		if (/\s/.test(typed)) {
			return null
		}
		const next = text.slice(caret).search(/\s/)
		return {kind, prefix: text[start], start, end: next === -1 ? text.length : caret + next, query: typed}
	}

	return null
}

// shortcut: a value with both quote types, or containing ` <prefix>`, cannot round-trip (the parser has no escapes); fine for real titles.
function quoted(value: string) {
	if (!/\s/.test(value) && !/^['"]/.test(value)) {
		return value
	}
	return value.includes('"') ? `'${value}'` : `"${value}"`
}

/** Replaces the token with `prefix + value + ' '` and returns the new text and the caret after it. */
export function insertToken(text: string, token: ActiveToken, value: string): {text: string, caret: number} {
	const inserted = token.prefix + quoted(value) + ' '
	// Reuse a space that already follows the token, so picking mid-text never doubles it.
	const rest = text.slice(token.end).replace(/^ /, '')
	return {
		text: text.slice(0, token.start) + inserted + rest,
		caret: token.start + inserted.length,
	}
}
