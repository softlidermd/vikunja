<template>
	<div class="filter-builder">
		<p
			v-if="isCustom"
			class="filter-builder__custom"
		>
			<Icon icon="circle-info" />
			<span>{{ $t('filters.builder.custom') }}</span>
		</p>

		<div
			v-else
			class="filter-builder__chips"
		>
			<Popup
				v-for="(condition, index) in conditions"
				:key="condition.field"
				placement="bottom-start"
				:anchor="chipEls[condition.field]"
				:open="openField === condition.field"
				sheet-on-mobile
				:sheet-title="fieldLabel(condition.field)"
				@update:open="open => setOpen(condition.field, open)"
			>
				<template #trigger="{toggle, isOpen}">
					<span
						:ref="el => { chipEls[condition.field] = el as HTMLElement | null }"
						class="filter-builder__chip"
						:class="{'is-open': isOpen}"
					>
						<BaseButton
							class="filter-builder__chip-label"
							:aria-expanded="isOpen"
							@click="toggle()"
						>
							{{ summary(condition) }}
							<Icon
								icon="chevron-down"
								class="filter-builder__chevron"
							/>
						</BaseButton>
						<BaseButton
							class="filter-builder__chip-remove"
							:aria-label="$t('filters.builder.remove', {field: fieldLabel(condition.field)})"
							@click="remove(index)"
						>
							<Icon icon="times" />
						</BaseButton>
					</span>
				</template>
				<template #content="{close}">
					<div
						class="filter-builder__panel"
						role="group"
						:aria-label="fieldLabel(condition.field)"
					>
						<template v-if="isMultiCondition(condition)">
							<div
								v-if="condition.field !== 'bucket_id'"
								class="filter-builder__segmented"
								role="group"
							>
								<BaseButton
									:class="{'is-selected': !condition.negate}"
									:aria-pressed="!condition.negate"
									@click="update(index, {negate: false})"
								>
									{{ $t('filters.builder.is') }}
								</BaseButton>
								<BaseButton
									:class="{'is-selected': condition.negate}"
									:aria-pressed="condition.negate"
									@click="update(index, {negate: true})"
								>
									{{ $t('filters.builder.isNot') }}
								</BaseButton>
							</div>
							<input
								v-model="search"
								class="input filter-builder__search"
								type="search"
								:aria-label="$t('filters.builder.search')"
								:placeholder="$t('filters.builder.search')"
							>
							<div class="filter-builder__options">
								<label
									v-for="option in optionsFor(condition)"
									:key="option.value"
									class="filter-builder__option"
								>
									<input
										type="checkbox"
										:checked="condition.values.includes(option.value)"
										@change="toggleValue(index, option.value)"
									>
									<span
										v-if="option.color"
										class="filter-builder__swatch"
										:style="{background: option.color}"
									/>
									<span>
										{{ option.title }}
										<span
											v-if="option.subtitle"
											class="filter-builder__subtitle"
										>{{ option.subtitle }}</span>
									</span>
								</label>
								<p
									v-if="optionsFor(condition).length === 0"
									class="filter-builder__subtitle"
								>
									{{ $t('filters.builder.noMatches') }}
								</p>
							</div>
							<p class="filter-builder__hint">
								{{ condition.negate ? $t('filters.builder.matchesNone') : $t('filters.builder.matchesAny') }}
							</p>
						</template>

						<fieldset
							v-else-if="condition.field === 'due_date'"
							class="filter-builder__fieldset"
						>
							<label
								v-for="preset in [...DUE_PRESETS, 'custom' as const]"
								:key="preset"
								class="filter-builder__option"
							>
								<input
									type="radio"
									:name="`filter-builder-due-${uid}`"
									:checked="condition.preset === preset"
									@change="update(index, {preset})"
								>
								{{ presetLabel(preset) }}
							</label>
							<div
								v-if="condition.preset === 'custom'"
								class="filter-builder__range"
							>
								<label>
									{{ $t('input.datepickerRange.from') }}
									<input
										class="input"
										type="date"
										:value="condition.from"
										@change="update(index, {from: ($event.target as HTMLInputElement).value})"
									>
								</label>
								<label>
									{{ $t('input.datepickerRange.to') }}
									<input
										class="input"
										type="date"
										:value="condition.to"
										@change="update(index, {to: ($event.target as HTMLInputElement).value})"
									>
								</label>
							</div>
						</fieldset>

						<template v-else-if="condition.field === 'priority'">
							<div
								class="filter-builder__segmented"
								role="group"
							>
								<BaseButton
									:class="{'is-selected': !condition.exact}"
									:aria-pressed="!condition.exact"
									@click="update(index, {exact: false})"
								>
									{{ $t('filters.builder.atLeast') }}
								</BaseButton>
								<BaseButton
									:class="{'is-selected': condition.exact}"
									:aria-pressed="condition.exact"
									@click="update(index, {exact: true})"
								>
									{{ $t('filters.builder.exactly') }}
								</BaseButton>
							</div>
							<div class="filter-builder__levels">
								<BaseButton
									v-for="level in PRIORITY_LEVELS"
									:key="level"
									class="filter-builder__level"
									:class="{'is-selected': isLevelSelected(condition, level)}"
									:aria-pressed="condition.value === level"
									@click="update(index, {value: condition.value === level ? null : level})"
								>
									{{ priorityLabel(level) }}
								</BaseButton>
							</div>
						</template>

						<fieldset
							v-else
							class="filter-builder__fieldset"
						>
							<label
								v-for="status in DONE_STATES"
								:key="String(status)"
								class="filter-builder__option"
							>
								<input
									type="radio"
									:name="`filter-builder-done-${uid}`"
									:checked="condition.value === status"
									@change="update(index, {value: status})"
								>
								{{ doneLabel(status) }}
							</label>
						</fieldset>

						<div class="filter-builder__footer">
							<XButton @click="close()">
								{{ $t('filters.builder.done') }}
							</XButton>
						</div>
					</div>
				</template>
			</Popup>

			<Popup
				v-if="addableFields.length > 0"
				placement="bottom-start"
				:anchor="addEl"
			>
				<template #trigger="{toggle, isOpen}">
					<BaseButton
						ref="addButton"
						class="filter-builder__add"
						:aria-expanded="isOpen"
						@click="toggle()"
					>
						<Icon icon="plus" />
						{{ $t('filters.builder.addFilter') }}
					</BaseButton>
				</template>
				<template #content="{close}">
					<div class="filter-builder__panel filter-builder__menu">
						<BaseButton
							v-for="field in addableFields"
							:key="field"
							class="filter-builder__menu-item"
							@click="close(); addField(field)"
						>
							{{ fieldLabel(field) }}
						</BaseButton>
					</div>
				</template>
			</Popup>

			<BaseButton
				v-if="currentUsername"
				class="filter-builder__me"
				:class="{'is-selected': isAssignedToMe}"
				:aria-pressed="isAssignedToMe"
				:disabled="isAssigneeNegated"
				@click="toggleAssignedToMe"
			>
				<Icon icon="user" />
				{{ $t('filters.builder.assignedToMe') }}
			</BaseButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, reactive, ref, useId, watch, type ComponentPublicInstance} from 'vue'
import {useI18n} from 'vue-i18n'
import {watchDebounced} from '@vueuse/core'
import {useQuery} from '@tanstack/vue-query'

import BaseButton from '@/components/base/BaseButton.vue'
import Popup from '@/components/misc/Popup.vue'
import type {User} from '@/client/generated'
import {searchProjectUsers, searchUsers} from '@/client/queries/userSearch'
import {useLabels} from '@/composables/useLabels'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {camelCase} from 'change-case'
import {PRIORITIES} from '@/constants/priorities'
import {PROJECT_VIEW_KINDS} from '@/constants/projectView'
import {bucketsQuery} from '@/client/queries/kanban'
import {transformFilterStringForApi} from '@/helpers/filters'
import {getStatusKanbanView} from '@/helpers/projectView'
import {
	BUILDER_FIELDS,
	DUE_PRESETS,
	emptyCondition,
	isMultiCondition,
	isPlainValue,
	looksLikeFilter,
	parseFilter,
	serializeCondition,
	serializeFilter,
	type BuilderField,
	type Condition,
	type DuePreset,
	type MultiCondition,
	type PriorityCondition,
} from '@/helpers/filterBuilder'

const props = withDefaults(defineProps<{
	modelValue?: string
	projectId?: number
	viewId?: number
}>(), {
	modelValue: '',
	projectId: undefined,
	viewId: undefined,
})

const emit = defineEmits<{
	'update:modelValue': [value: string],
}>()

const {t} = useI18n({useScope: 'global'})
const uid = useId()

const PRIORITY_NAMES = Object.fromEntries(Object.entries(PRIORITIES).map(([name, level]) => [level, camelCase(name)]))
const PRIORITY_LEVELS = Object.values(PRIORITIES).filter(level => level > PRIORITIES.UNSET)
const DONE_STATES = [null, false, true] as const

const conditions = ref<Condition[]>([])
const isCustom = ref(false)
const openField = ref<BuilderField | null>(null)
const search = ref('')

const {labels, getLabelById, getLabelByExactTitle} = useLabels()
const projectList = useProjects()

// View settings hold titles and camelCase fields, FilterInput emits ids and snake_case: parse both as the latter.
// A number is already an id; resolving it as a title would pick a label or project named "4" instead.
function toApiForm(filter: string): string {
	const isId = (value: string) => /^\d+$/.test(value)
	return transformFilterStringForApi(
		filter,
		title => isId(title) ? null : getLabelByExactTitle(title)?.id || null,
		title => isId(title) ? null : projectList.findProjectByExactname(title)?.id || null,
	)
}

// Only rebuild the chips when the query changes in meaning; an echo of our own emit (or a reformatted copy of it)
// must not reset the chip that is being edited.
watch(
	() => toApiForm(props.modelValue ?? ''),
	value => {
		// Plain text without filter syntax is a title search: no chips, but nothing to warn about either.
		const parsed = looksLikeFilter(value) ? parseFilter(value) : []
		if (parsed !== null && serializeFilter(parsed) === serializeFilter(conditions.value) && !isCustom.value) {
			return
		}
		isCustom.value = parsed === null
		conditions.value = parsed ?? []
		openField.value = null
	},
	{immediate: true},
)

function commit() {
	emit('update:modelValue', serializeFilter(conditions.value))
}

function update(index: number, patch: Partial<Condition>) {
	conditions.value[index] = {...conditions.value[index], ...patch} as Condition
	commit()
}

function toggleValue(index: number, value: string) {
	const condition = conditions.value[index] as MultiCondition
	const values = condition.values.includes(value)
		? condition.values.filter(v => v !== value)
		: [...condition.values, value]
	update(index, {values})
}

function remove(index: number) {
	conditions.value.splice(index, 1)
	openField.value = null
	commit()
}

function pruneEmpty() {
	conditions.value = conditions.value.filter(condition => serializeCondition(condition) !== '')
}

function setOpen(field: BuilderField, open: boolean) {
	if (open) {
		openField.value = field
		search.value = ''
		return
	}
	if (openField.value === field) {
		openField.value = null
		pruneEmpty()
	}
}

function addField(field: BuilderField) {
	conditions.value.push(emptyCondition(field))
	openField.value = field
	search.value = ''
	commit()
}

const projectViews = computed(() => projectList.projects[props.projectId ?? 0]?.views)
const statusView = computed(() => getStatusKanbanView(projectViews.value, props.viewId))
// Any bucket_id in a Kanban view's filter makes the API return a flat list instead of the board.
const canFilterStatus = computed(() => statusView.value !== null && props.viewId !== undefined &&
	projectViews.value?.find(view => view.id === props.viewId)?.view_kind !== PROJECT_VIEW_KINDS.KANBAN)
// Loaded even where the chip can't be added, so a status filter from elsewhere shows bucket names, not ids.
const statusBuckets = useQuery(computed(() => bucketsQuery(props.projectId ?? 0, statusView.value?.id ?? 0)))

const addableFields = computed(() => BUILDER_FIELDS.filter(field =>
	!conditions.value.some(c => c.field === field) && (field !== 'bucket_id' || canFilterStatus.value)))

const addButton = ref<ComponentPublicInstance | null>(null)
const addEl = computed(() => (addButton.value?.$el as HTMLElement | undefined) ?? null)
const chipEls = reactive<Partial<Record<BuilderField, HTMLElement | null>>>({})

const authStore = useAuthStore()
const currentUsername = computed(() => authStore.info?.username ?? '')
// Toggling "me" on a negated assignee chip would silently drop the exclusion.
const isAssigneeNegated = computed(() => conditions.value.some(c => c.field === 'assignees' && c.negate))
const isAssignedToMe = computed(() => conditions.value.some(c =>
	c.field === 'assignees' && !c.negate && c.values.includes(currentUsername.value),
))

function toggleAssignedToMe() {
	const index = conditions.value.findIndex(c => c.field === 'assignees')
	if (index === -1) {
		conditions.value.push({field: 'assignees', negate: false, values: [currentUsername.value]})
		commit()
		return
	}
	const condition = conditions.value[index] as MultiCondition
	if (isAssignedToMe.value) {
		const values = condition.values.filter(v => v !== currentUsername.value)
		if (values.length === 0) {
			remove(index)
			return
		}
		update(index, {values})
		return
	}
	update(index, {values: [...condition.values, currentUsername.value]})
}

const userResults = ref<User[]>([])
const userNames = reactive<Record<string, string>>({})

watchDebounced(
	[search, openField],
	async ([query, field]) => {
		if (field !== 'assignees' && field !== 'created_by') {
			return
		}
		try {
			// Saved filters route with a negative pseudo project id, which has no members to search.
			const found = props.projectId && props.projectId > 0
				? await searchProjectUsers(props.projectId, query)
				: await searchUsers(query)
			if (query !== search.value || field !== openField.value) {
				return
			}
			userResults.value = found
			found.forEach(user => {
				if (user.username) {
					userNames[user.username] = user.name || user.username
				}
			})
		} catch {
			userResults.value = []
		}
	},
	{debounce: 200},
)

type Option = {value: string, title: string, subtitle?: string, color?: string}

function displayValue(field: BuilderField, value: string): string {
	if (field === 'labels') {
		return getLabelById(Number(value))?.title ?? value
	}
	if (field === 'project') {
		return projectList.projects[Number(value)]?.title ?? value
	}
	if (field === 'bucket_id') {
		return statusBuckets.data.value?.find(bucket => bucket.id === Number(value))?.title ?? value
	}
	return userNames[value] ?? value
}

function optionsFor(condition: MultiCondition): Option[] {
	const query = search.value.toLowerCase()
	let options: Option[]
	if (condition.field === 'labels') {
		options = labels.value.map(label => ({
			value: String(label.id),
			title: label.title ?? '',
			color: label.hex_color ? '#' + label.hex_color.replace('#', '') : undefined,
		}))
	} else if (condition.field === 'project') {
		// Negative ids are pseudo projects (saved filters, favorites).
		options = projectList.projectsArray
			.filter(project => project.id > 0)
			.map(project => ({value: String(project.id), title: project.title ?? ''}))
	} else if (condition.field === 'bucket_id') {
		options = (statusBuckets.data.value ?? []).map(bucket => ({value: String(bucket.id), title: bucket.title ?? ''}))
	} else {
		// The server already filtered users by the search text.
		options = userResults.value
			.filter(user => user.username && isPlainValue(user.username))
			.map(user => ({
				value: user.username!,
				title: user.name || user.username!,
				subtitle: user.name ? '@' + user.username : undefined,
			}))
	}
	if (condition.field === 'labels' || condition.field === 'project' || condition.field === 'bucket_id') {
		options = options.filter(option => option.title.toLowerCase().includes(query))
	}
	// Keep selected values listed even when the search or the user lookup no longer returns them.
	const missing = condition.values
		.filter(value => !options.some(option => option.value === value))
		.map(value => ({value, title: displayValue(condition.field, value)}))
	return [...missing, ...options]
}

const SHARED_FIELD_LABELS: Partial<Record<BuilderField, string>> = {
	priority: 'task.attributes.priority',
	project: 'task.attributes.project',
}

function fieldLabel(field: BuilderField): string {
	return t(SHARED_FIELD_LABELS[field] ?? `filters.builder.fields.${field}`)
}

function priorityLabel(level: number): string {
	return t(`task.priority.${PRIORITY_NAMES[level]}`)
}

function presetLabel(preset: DuePreset | 'custom'): string {
	if (preset === 'overdue') return t('filters.builder.overdue')
	if (preset === 'custom') return t('filters.builder.customRange')
	return t(`input.datepickerRange.ranges.${preset}`)
}

function doneLabel(value: boolean | null): string {
	if (value === null) return t('filters.builder.statusAny')
	return value ? t('task.attributes.done') : t('filters.builder.statusNotDone')
}

function isLevelSelected(condition: PriorityCondition, level: number): boolean {
	if (condition.value === null) return false
	return condition.exact ? level === condition.value : level >= condition.value
}

function formatDay(date: string): string {
	return date
		? new Date(date + 'T00:00:00Z').toLocaleDateString(undefined, {month: 'short', day: 'numeric', timeZone: 'UTC'})
		: '…'
}

function summary(condition: Condition): string {
	const field = fieldLabel(condition.field)
	if (isMultiCondition(condition)) {
		if (condition.values.length === 0) {
			return t('filters.builder.summary', {field, value: t('filters.builder.any')})
		}
		const names = condition.values.map(value => displayValue(condition.field, value))
		const more = names.length > 2 ? ' ' + t('filters.builder.more', {count: names.length - 2}) : ''
		const value = names.slice(0, 2).join(', ') + more
		return t(condition.negate ? 'filters.builder.summaryNot' : 'filters.builder.summary', {field, value})
	}
	if (condition.field === 'due_date') {
		const value = condition.preset === null
			? t('filters.builder.any')
			: condition.preset === 'custom'
				? `${formatDay(condition.from)} – ${formatDay(condition.to)}`
				: presetLabel(condition.preset)
		return t('filters.builder.summary', {field, value})
	}
	if (condition.field === 'priority') {
		const value = condition.value === null
			? t('filters.builder.any')
			: condition.exact
				? priorityLabel(condition.value)
				: t('filters.builder.orAbove', {priority: priorityLabel(condition.value)})
		return t('filters.builder.summary', {field, value})
	}
	return t('filters.builder.summary', {field, value: doneLabel(condition.value)})
}
</script>

<style lang="scss" scoped>
.filter-builder {
	margin-block-end: .75rem;
}

.filter-builder__chips {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
}

.filter-builder__chip {
	display: inline-flex;
	align-items: center;
	border: 1px solid var(--primary);
	border-radius: 999px;
	background: var(--primary-light);
	overflow: hidden;

	&.is-open {
		box-shadow: 0 0 0 2px var(--primary);
	}
}

.filter-builder__chip-label {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	min-block-size: 2.25rem;
	padding-inline: .875rem .25rem;
	color: var(--primary-dark);
	font-weight: 600;
}

.filter-builder__chevron {
	font-size: .75em;
}

.filter-builder__chip-remove {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	min-inline-size: 2rem;
	min-block-size: 2.25rem;
	color: var(--primary-dark);
}

.filter-builder__add,
.filter-builder__me {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	min-block-size: 2.25rem;
	padding-inline: .875rem;
	border-radius: 999px;
	font-weight: 600;
}

.filter-builder__add {
	border: 1px dashed var(--grey-400);
	color: var(--grey-700);
}

.filter-builder__me {
	border: 1px solid var(--grey-300);
	color: var(--grey-700);

	&.is-selected {
		border-color: var(--primary);
		background: var(--primary);
		color: var(--white);
	}
}

.filter-builder__custom {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
	padding: .75rem;
	border-radius: $radius;
	background: var(--warning-light);
	color: var(--grey-800);
}

.filter-builder__panel {
	display: flex;
	flex-direction: column;
	gap: .625rem;
	inline-size: 18.75rem;
	max-inline-size: 100%;
	padding: .75rem;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-md);
}

.filter-builder__menu {
	inline-size: auto;
	min-inline-size: 12rem;
	gap: 0;
	padding: .375rem;
}

.filter-builder__menu-item {
	padding: .5rem .75rem;
	border-radius: $radius;
	text-align: start;

	&:hover,
	&:focus-visible {
		background: var(--grey-100);
	}
}

.filter-builder__segmented {
	display: flex;
	gap: .25rem;
	padding: .2rem;
	border-radius: $radius;
	background: var(--grey-100);

	> * {
		flex: 1;
		min-block-size: 2rem;
		border-radius: $radius;
		font-weight: 600;
		color: var(--grey-600);
	}

	.is-selected {
		background: var(--white);
		color: var(--grey-900);
		box-shadow: var(--shadow-sm);
	}
}

.filter-builder__options {
	display: flex;
	flex-direction: column;
	max-block-size: 14rem;
	overflow-y: auto;
}

.filter-builder__option {
	display: flex;
	align-items: center;
	gap: .625rem;
	min-block-size: 2.25rem;
	padding-inline: .5rem;
	border-radius: $radius;
	cursor: pointer;

	&:hover {
		background: var(--grey-100);
	}
}

.filter-builder__swatch {
	flex: none;
	inline-size: .75rem;
	block-size: .75rem;
	border-radius: 3px;
}

.filter-builder__subtitle {
	display: block;
	font-size: .75rem;
	color: var(--grey-500);
}

.filter-builder__hint {
	font-size: .75rem;
	color: var(--grey-500);
}

.filter-builder__fieldset {
	display: flex;
	flex-direction: column;
	border: 0;
	padding: 0;
	margin: 0;
}

.filter-builder__range {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: .5rem;
	padding: .375rem .5rem 0;
	font-size: .75rem;
	color: var(--grey-600);
}

.filter-builder__levels {
	display: flex;
	flex-wrap: wrap;
	gap: .25rem;
}

.filter-builder__level {
	min-block-size: 2rem;
	padding-inline: .625rem;
	border: 1px solid var(--grey-300);
	border-radius: $radius;
	font-size: .875rem;
	font-weight: 600;

	&.is-selected {
		border-color: var(--primary);
		background: var(--primary);
		color: var(--white);
	}
}

.filter-builder__footer {
	display: flex;
	justify-content: flex-end;
}
</style>
