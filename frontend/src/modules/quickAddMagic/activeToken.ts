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

// prefixParser splits on ' ' + prefix within a line, so a tab is part of the value, not a boundary.
const isBoundary = (char: string) => char === ' ' || char === '\n'

/**
 * Finds the label, assignee or project token the caret is in, by the rules of prefixParser:
 * the prefix starts the line or follows a space, and the value runs to the next space
 * or, when it starts with a quote, to the matching quote (or the caret while it is unclosed).
 */
export function getActiveToken(text: string, caret: number, prefixes: Prefixes | undefined): ActiveToken | null {
	if (!prefixes) {
		return null
	}

	for (let start = caret - 1; start >= 0 && text[start] !== '\n'; start--) {
		const kind = KINDS.find(k => prefixes[k] === text[start])
		if (!kind || (start > 0 && !isBoundary(text[start - 1]))) {
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
			// Without a closing quote the token ends at the caret, so a pick keeps the words after it.
			const end = close !== -1 && (lineEnd === -1 || close < lineEnd) ? close + 1 : caret
			return {kind, prefix: text[start], start, end, query: typed.slice(1)}
		}

		if ([...typed].some(isBoundary)) {
			return null
		}
		const next = text.slice(caret).search(/[ \n]/)
		return {kind, prefix: text[start], start, end: next === -1 ? text.length : caret + next, query: typed}
	}

	return null
}

/** Keeps the items whose title contains the query: exact matches first, then prefixes, then the rest. */
export function rankByTitle<T>(items: T[], query: string, title: (item: T) => string): T[] {
	const q = query.toLowerCase()
	const rank = (item: T) => {
		const value = title(item).toLowerCase()
		if (value === q) return 0
		return value.startsWith(q) ? 1 : 2
	}
	return items
		.filter(item => title(item).toLowerCase().includes(q))
		.sort((a, b) => rank(a) - rank(b))
}

// shortcut: a value with both quote types, or containing ` <prefix>`, cannot round-trip
// (the parser has no escapes); fine for real titles.
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
