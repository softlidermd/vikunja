import {snakeCase} from 'change-case'
import {DATE_RANGES} from '@/components/date/dateRanges'
import {AVAILABLE_FILTER_FIELDS, FILTER_OPERATORS_REGEX} from '@/helpers/filters'

// A view over the API-form filter string: a flat AND of the conditions below, anything else parses to null.

export type MultiField = 'assignees' | 'labels' | 'project' | 'created_by'
export type BuilderField = MultiField | 'due_date' | 'priority' | 'done'

export const DUE_PRESETS = [
	'overdue',
	'today',
	'tomorrow',
	'thisWeek',
	'nextWeek',
	'next7Days',
	'thisMonth',
	'next30Days',
] as const
export type DuePreset = typeof DUE_PRESETS[number]

export type MultiCondition = {field: MultiField, negate: boolean, values: string[]}
export type DueCondition = {field: 'due_date', preset: DuePreset | 'custom' | null, from: string, to: string}
export type PriorityCondition = {field: 'priority', value: number | null, exact: boolean}
export type DoneCondition = {field: 'done', value: boolean | null}
export type Condition = MultiCondition | DueCondition | PriorityCondition | DoneCondition

export const BUILDER_FIELDS: BuilderField[] = [
	'assignees',
	'labels',
	'due_date',
	'priority',
	'done',
	'project',
	'created_by',
]
const MULTI_FIELDS = new Set<BuilderField>(['assignees', 'labels', 'project', 'created_by'])

export function isMultiCondition(condition: Condition): condition is MultiCondition {
	return MULTI_FIELDS.has(condition.field)
}

export function emptyCondition(field: BuilderField): Condition {
	switch (field) {
		case 'due_date': return {field, preset: null, from: '', to: ''}
		case 'priority': return {field, value: null, exact: false}
		case 'done': return {field, value: false}
		default: return {field, negate: false, values: []}
	}
}

const FIELD_ALIASES: Record<string, BuilderField> = {
	assignees: 'assignees',
	labels: 'labels',
	project: 'project',
	created_by: 'created_by',
	createdBy: 'created_by',
	due_date: 'due_date',
	dueDate: 'due_date',
	priority: 'priority',
	done: 'done',
}

type Clause = [operator: string, value: string]

function dueClauses(preset: DuePreset): Clause[] {
	if (preset === 'overdue') {
		return [['<', 'now']]
	}
	const [from, to] = DATE_RANGES[preset]
	return [['>=', from], ['<', to]]
}

function isIsoDate(value: string): boolean {
	const date = new Date(value + 'T00:00:00Z')
	return /^\d{4}-\d{2}-\d{2}$/.test(value) && !isNaN(date.getTime()) && date.toISOString().startsWith(value)
}

function addDays(date: string, days: number): string {
	const d = new Date(date + 'T00:00:00Z')
	d.setUTCDate(d.getUTCDate() + days)
	return d.toISOString().slice(0, 10)
}

type Token = {type: 'word' | 'string' | 'op' | 'comma', value: string}

// Longest first, so ">=" is never read as ">" followed by "=".
const OPERATORS = ['&&', '||', '>=', '<=', '!=', '?=', '=', '>', '<', '(', ')']
const WORD_BREAK = /[\s,"'&|()=!<>?]/

// Values the backend splits or cuts at (spaces, commas, &, |, brackets) can't be written into a list.
export function isPlainValue(value: string): boolean {
	return !WORD_BREAK.test(value)
}

const FIELD_NAMES = AVAILABLE_FILTER_FIELDS.flatMap(field => [field, snakeCase(field)])
const FIELD_CLAUSE = new RegExp('\\b(' + FIELD_NAMES.join('|') + ')\\s*' + FILTER_OPERATORS_REGEX, 'i')

// Unlike hasFilterQuery, a search such as "planning" or "in progress" is not mistaken for a filter.
export function looksLikeFilter(filter: string): boolean {
	return FIELD_CLAUSE.test(filter)
}

function tokenize(input: string): Token[] | null {
	const tokens: Token[] = []
	let i = 0
	while (i < input.length) {
		const ch = input[i]!
		if (/\s/.test(ch)) {
			i++
			continue
		}
		if (ch === ',') {
			tokens.push({type: 'comma', value: ch})
			i++
			continue
		}
		if (ch === '"' || ch === '\'') {
			let j = i + 1
			let value = ''
			while (j < input.length && input[j] !== ch) {
				if (input[j] === '\\' && j + 1 < input.length) {
					value += input[j + 1]
					j += 2
					continue
				}
				value += input[j]
				j++
			}
			if (j >= input.length) {
				return null
			}
			tokens.push({type: 'string', value})
			i = j + 1
			continue
		}
		const op = OPERATORS.find(o => input.startsWith(o, i))
		if (op) {
			tokens.push({type: 'op', value: op})
			i += op.length
			continue
		}
		let j = i
		while (j < input.length && !WORD_BREAK.test(input[j]!)) {
			j++
		}
		if (j === i) {
			return null
		}
		tokens.push({type: 'word', value: input.slice(i, j)})
		i = j
	}
	return tokens
}

function parseClause(tokens: Token[]): {field: BuilderField, operator: string, values: string[]} | null {
	const [fieldToken, ...rest] = tokens
	if (fieldToken?.type !== 'word' || !(fieldToken.value in FIELD_ALIASES)) {
		return null
	}

	let operator: string
	if (rest[0]?.type === 'op') {
		operator = rest.shift()!.value
	} else if (rest[0]?.type === 'word' && rest[0].value === 'in') {
		rest.shift()
		operator = 'in'
	} else if (rest[0]?.type === 'word' && rest[0].value === 'not' && rest[1]?.type === 'word' && rest[1].value === 'in') {
		rest.splice(0, 2)
		operator = 'not in'
	} else {
		return null
	}

	const values: string[] = []
	for (let i = 0; i < rest.length; i++) {
		const token = rest[i]!
		const expectValue = i % 2 === 0
		if (expectValue && (token.type === 'word' || token.type === 'string')) {
			values.push(token.value)
		} else if (!expectValue && token.type === 'comma') {
			continue
		} else {
			return null
		}
	}
	if (values.length === 0 || rest.at(-1)?.type === 'comma') {
		return null
	}

	return {field: FIELD_ALIASES[fieldToken.value]!, operator, values}
}

export function parseFilter(filter: string): Condition[] | null {
	const tokens = tokenize(filter)
	if (tokens === null) {
		return null
	}
	if (tokens.length === 0) {
		return []
	}

	const clauses: Token[][] = [[]]
	for (const token of tokens) {
		if (token.type === 'op' && ['||', '(', ')'].includes(token.value)) {
			return null
		}
		if (token.type === 'op' && token.value === '&&') {
			clauses.push([])
			continue
		}
		clauses.at(-1)!.push(token)
	}

	const conditions: Condition[] = []
	const seen = new Set<BuilderField>()
	const due: Clause[] = []

	for (const tokensOfClause of clauses) {
		const clause = parseClause(tokensOfClause)
		if (clause === null) {
			return null
		}
		const {field, operator, values} = clause

		if (field === 'due_date') {
			if (values.length !== 1) {
				return null
			}
			if (due.length === 0) {
				conditions.push(emptyCondition('due_date'))
			}
			due.push([operator, values[0]!])
			continue
		}

		// One chip per field: a repeated field (labels = 1 && labels = 2) means "both", which a chip can't show.
		if (seen.has(field)) {
			return null
		}
		seen.add(field)

		if (field === 'priority') {
			const value = Number(values[0])
			const validLevel = Number.isInteger(value) && value >= 1 && value <= 5
			if (values.length !== 1 || !(operator === '>=' || operator === '=') || !validLevel) {
				return null
			}
			conditions.push({field, value, exact: operator === '='})
			continue
		}

		if (field === 'done') {
			if (values.length !== 1 || operator !== '=' || !['true', 'false'].includes(values[0]!)) {
				return null
			}
			conditions.push({field, value: values[0] === 'true'})
			continue
		}

		// The backend splits list values on commas and keeps quotes literally, so only plain words round-trip.
		if (!values.every(isPlainValue)) {
			return null
		}
		if (operator === 'in' || operator === '?=' || (operator === '=' && values.length === 1)) {
			conditions.push({field, negate: false, values})
		} else if (operator === 'not in') {
			conditions.push({field, negate: true, values})
		} else {
			return null
		}
	}

	if (due.length > 0) {
		const condition = conditions.find((c): c is DueCondition => c.field === 'due_date')!
		const preset = DUE_PRESETS.find(p => {
			const expected = dueClauses(p)
			return expected.length === due.length &&
				expected.every(([op, value]) => due.some(([o, v]) => o === op && v === value))
		})
		if (preset) {
			condition.preset = preset
		} else {
			condition.preset = 'custom'
			for (const [operator, value] of due) {
				if (!isIsoDate(value)) {
					return null
				}
				if (operator === '>=' && condition.from === '') {
					condition.from = value
				} else if (operator === '<' && condition.to === '') {
					// "To" is inclusive in the UI; the query holds the exclusive next day.
					condition.to = addDays(value, -1)
				} else {
					return null
				}
			}
		}
	}

	return conditions
}

export function serializeCondition(condition: Condition): string {
	switch (condition.field) {
		case 'due_date': {
			const {preset, from, to} = condition
			if (preset === null) {
				return ''
			}
			const clauses: Clause[] = preset === 'custom'
				? [...(from ? [['>=', from] as Clause] : []), ...(to ? [['<', addDays(to, 1)] as Clause] : [])]
				: dueClauses(preset)
			return clauses.map(([op, value]) => `due_date ${op} ${value}`).join(' && ')
		}
		case 'priority':
			return condition.value === null ? '' : `priority ${condition.exact ? '=' : '>='} ${condition.value}`
		case 'done':
			return condition.value === null ? '' : `done = ${condition.value}`
		default:
			return condition.values.length === 0
				? ''
				: `${condition.field} ${condition.negate ? 'not in' : 'in'} ${condition.values.join(', ')}`
	}
}

export function serializeFilter(conditions: Condition[]): string {
	return conditions.map(serializeCondition).filter(part => part !== '').join(' && ')
}
