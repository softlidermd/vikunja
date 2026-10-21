import {beforeEach, describe, expect, it} from 'vitest'
import {createPinia, setActivePinia} from 'pinia'

import {getActiveToken, insertToken, parseTaskText, PREFIXES, PrefixMode, rankByTitle} from '.'

const vikunja = PREFIXES[PrefixMode.Default]
const todoist = PREFIXES[PrefixMode.Todoist]

// The caret sits where `|` is.
function tokenAt(textWithCaret: string, prefixes = vikunja) {
	const caret = textWithCaret.indexOf('|')
	return getActiveToken(textWithCaret.replace('|', ''), caret, prefixes)
}

function pick(textWithCaret: string, value: string, prefixes = vikunja) {
	const token = tokenAt(textWithCaret, prefixes)
	expect(token).not.toBeNull()
	return insertToken(textWithCaret.replace('|', ''), token!, value)
}

describe('getActiveToken', () => {
	it.each([
		['Task *urg|', vikunja, 'label', '*'],
		['Task @ali|', vikunja, 'assignee', '@'],
		['Task +Hom|', vikunja, 'project', '+'],
		['Task @urg|', todoist, 'label', '@'],
		['Task +ali|', todoist, 'assignee', '+'],
		['Task #Hom|', todoist, 'project', '#'],
	])('finds the %s token', (text, prefixes, kind, prefix) => {
		expect(tokenAt(text, prefixes)).toMatchObject({kind, prefix, query: text.slice(6, -1)})
	})

	it('finds a bare prefix at the start of the text', () => {
		expect(tokenAt('*|')).toEqual({kind: 'label', prefix: '*', start: 0, end: 1, query: ''})
	})

	it('ignores the priority prefix', () => {
		expect(tokenAt('Task !3|')).toBeNull()
	})

	it('finds nothing when magic is disabled', () => {
		expect(getActiveToken('Task *urg', 9, PREFIXES[PrefixMode.Disabled])).toBeNull()
	})

	it('needs whitespace before the prefix', () => {
		expect(tokenAt('a@b|')).toBeNull()
	})

	it('treats a tab like the parser does: not a boundary', () => {
		expect(tokenAt('Task\t*urg|')).toBeNull()
		expect(tokenAt('Task *ur|g\tent rest')).toMatchObject({end: 13})
	})

	it('finds a prefix after a line break', () => {
		expect(tokenAt('first\n*urg|')).toMatchObject({kind: 'label', start: 6, query: 'urg'})
	})

	it('closes after a space following an unquoted token', () => {
		expect(tokenAt('Task *urgent |')).toBeNull()
	})

	it('stays active inside an open quote', () => {
		expect(tokenAt('Task *"Hello Wo|')).toMatchObject({query: 'Hello Wo', start: 5, end: 15})
	})

	it('ends an unclosed quote at the caret, keeping the words after it', () => {
		const text = 'Купить *"Распо| молоко завтра'
		expect(tokenAt(text)).toMatchObject({start: 7, end: 14, query: 'Распо'})
		expect(pick(text, 'Распознавание и сопоставление').text)
			.toBe('Купить *"Распознавание и сопоставление" молоко завтра')
	})

	it('runs end through the closing quote with the caret inside', () => {
		const text = 'Task *"Hello World" rest'
		expect(getActiveToken(text, text.indexOf(' World'), vikunja)).toMatchObject({start: 5, end: 19, query: 'Hello'})
	})

	it('closes after the closing quote', () => {
		expect(tokenAt('Task *"Hello World"|')).toBeNull()
	})

	it('runs an unquoted end to the next whitespace', () => {
		expect(tokenAt('Task *ur|gent rest')).toMatchObject({start: 5, end: 12, query: 'ur'})
	})
})

describe('rankByTitle', () => {
	const rank = (titles: string[], query: string) => rankByTitle(titles, query, title => title)

	it('puts an exact match first, then prefixes, then the rest', () => {
		expect(rank(['non-urgent', 'Urgently', 'urgent', 'other'], 'URGENT'))
			.toEqual(['urgent', 'Urgently', 'non-urgent'])
	})

	it('keeps the original order within a rank', () => {
		expect(rank(['b-x', 'a-x', 'x'], '')).toEqual(['b-x', 'a-x', 'x'])
	})
})

describe('insertToken', () => {
	it('replaces the token and puts the caret after the space', () => {
		expect(pick('Task *urg|', 'urgent')).toEqual({text: 'Task *urgent ', caret: 13})
	})

	it('keeps the rest of the text when replacing mid-text', () => {
		const result = pick('Buy *mi|lk tomorrow', 'shopping')
		expect(result.text).toBe('Buy *shopping tomorrow')
		expect(result.caret).toBe('Buy *shopping '.length)
	})

	it('replaces a quoted token through its closing quote', () => {
		const text = 'Task *"Hello World" rest'
		const token = getActiveToken(text, 8, vikunja)!
		expect(insertToken(text, token, 'Hello Kitty').text).toBe('Task *"Hello Kitty" rest')
	})

	it.each([
		['Hello World', '*"Hello World" '],
		['"quoted', '*\'"quoted\' '],
		['\'single', '*"\'single" '],
		['Say "hi" now', '*\'Say "hi" now\' '],
		['plain', '*plain '],
	])('quotes %s when needed', (value, expected) => {
		expect(pick('*|', value).text).toBe(expected)
	})
})

describe('round trips through parseTaskText', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
	})

	it('keeps a label with spaces in Vikunja mode', () => {
		const text = pick('Task *Расп|', 'Распознавание и сопоставление').text
		const parsed = parseTaskText(text, PrefixMode.Default)
		expect(parsed.labels).toEqual(['Распознавание и сопоставление'])
		expect(parsed.text).toBe('Task')
	})

	it('keeps a project title with spaces', () => {
		const parsed = parseTaskText(pick('Task +My|', 'My Project').text,PrefixMode.Default)
		expect(parsed.project).toBe('My Project')
		expect(parsed.text).toBe('Task')
	})

	it('keeps a Todoist label with spaces', () => {
		const parsed = parseTaskText(pick('Task @Hel| later', 'Hello World', todoist).text, PrefixMode.Todoist)
		expect(parsed.labels).toEqual(['Hello World'])
		expect(parsed.text).toBe('Task later')
	})

	it('keeps an assignee username', () => {
		const parsed = parseTaskText(pick('Task @al|', 'alice').text, PrefixMode.Default)
		expect(parsed.assignees).toEqual(['alice'])
	})
})
