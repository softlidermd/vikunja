import {describe, expect, it} from 'vitest'
import {isPlainValue, looksLikeFilter, parseFilter, serializeFilter, type Condition} from './filterBuilder'

describe('filter builder', () => {
	it('tells a title search from a filter', () => {
		for (const search of ['planning', 'in progress', 'meeting notes', 'like a boss']) {
			expect(looksLikeFilter(search), search).toBe(false)
		}
		const filters = ['labels in 1', 'due_date < now', 'dueDate<now', 'done = false', 'priority >= 3 || done = true']
		for (const filter of filters) {
			expect(looksLikeFilter(filter), filter).toBe(true)
		}
	})

	it('only accepts list values the backend does not split', () => {
		expect(isPlainValue('alice')).toBe(true)
		for (const value of ['a&b', 'a|b', 'a(b)', 'bob smith', 'a,b']) {
			expect(isPlainValue(value), value).toBe(false)
		}
	})

	it('parses a flat AND of supported conditions', () => {
		expect(parseFilter('assignees in alice, bob && labels not in 3 && done = false && priority >= 3')).toEqual([
			{field: 'assignees', negate: false, values: ['alice', 'bob']},
			{field: 'labels', negate: true, values: ['3']},
			{field: 'done', value: false},
			{field: 'priority', value: 3, exact: false},
		])
	})

	it('reads equivalent spellings as the same chip', () => {
		expect(parseFilter('createdBy = alice && project ?= 4 && priority=5')).toEqual([
			{field: 'created_by', negate: false, values: ['alice']},
			{field: 'project', negate: false, values: ['4']},
			{field: 'priority', value: 5, exact: true},
		])
	})

	it('recognises date presets in either clause order, and absolute ranges', () => {
		expect(parseFilter('due_date < now')).toEqual([{field: 'due_date', preset: 'overdue', from: '', to: ''}])
		expect(parseFilter('due_date < now/w+1w && due_date >= now/w')).toEqual([
			{field: 'due_date', preset: 'thisWeek', from: '', to: ''},
		])
		expect(parseFilter('due_date >= 2026-10-01 && due_date < 2026-11-01')).toEqual([
			{field: 'due_date', preset: 'custom', from: '2026-10-01', to: '2026-10-31'},
		])
	})

	it('returns null for anything a chip cannot show', () => {
		for (const query of [
			'priority = 5 || done = true',
			'(done = false)',
			'labels = 1 && labels = 2',
			'percent_done > 50',
			'labels like 1',
			'priority > 2',
			'due_date > now',
			'due_date >= now/d && due_date >= now/w',
			'due_date < 2026-13-01',
			'due_date >= 2026-02-30',
			'labels in "a,b"',
			'assignees in "bob smith"',
			'just some search text',
			'assignees in "unclosed',
			'labels in 1,',
		]) {
			expect(parseFilter(query), query).toBeNull()
		}
	})

	it('serializes back to a query that parses to the same conditions', () => {
		const conditions: Condition[] = [
			{field: 'assignees', negate: false, values: ['alice', 'bob']},
			{field: 'due_date', preset: 'custom', from: '2026-10-01', to: '2026-10-31'},
			{field: 'priority', value: 4, exact: true},
			{field: 'done', value: true},
		]
		const query = serializeFilter(conditions)
		expect(query).toBe(
			'assignees in alice, bob && due_date >= 2026-10-01 && due_date < 2026-11-01 && priority = 4 && done = true',
		)
		expect(parseFilter(query)).toEqual(conditions)
	})

	it('drops conditions that have nothing selected', () => {
		expect(serializeFilter([
			{field: 'labels', negate: false, values: []},
			{field: 'due_date', preset: null, from: '', to: ''},
			{field: 'done', value: null},
			{field: 'priority', value: 2, exact: false},
		])).toBe('priority >= 2')
		expect(parseFilter('')).toEqual([])
	})
})
