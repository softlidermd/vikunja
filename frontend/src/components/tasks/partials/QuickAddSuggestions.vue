<template>
	<Teleport to="body">
		<div
			v-if="isOpen"
			:id="listId"
			ref="list"
			class="quick-add-suggestions"
			role="listbox"
			:aria-label="$t('task.quickAddSuggest.label')"
			@pointerdown.prevent
			@mousedown.prevent
		>
			<div
				v-for="row in rows"
				:id="row.value ? `${listId}-${options.indexOf(row)}` : undefined"
				:key="row.key"
				class="suggestion"
				:class="{'is-selected': row === selected, 'is-inert': !row.value}"
				:role="row.value ? 'option' : undefined"
				:aria-selected="row.value ? row === selected : undefined"
				tabindex="-1"
				@focus="input?.focus()"
				@click="row.value && pick(row.value)"
			>
				<span
					v-if="row.label"
					class="tag"
					:style="getLabelStyles(row.label)"
				>{{ row.label.title }}</span>
				<User
					v-else-if="row.user"
					:user="row.user"
					:avatar-size="24"
					:show-username="!row.text"
				/>
				<Icon
					v-else-if="row.isNewLabel"
					icon="plus"
					class="create-icon"
				/>
				<span
					v-if="row.text"
					class="suggestion-text"
				>{{ row.text }}</span>
				<span
					v-if="row.hint"
					class="hint-text"
				>{{ row.hint }}</span>
			</div>
		</div>
	</Teleport>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, useId, watch, watchEffect} from 'vue'
import {useI18n} from 'vue-i18n'
import {refDebounced, useEventListener} from '@vueuse/core'
import {autoUpdate, computePosition, flip, offset, shift} from '@floating-ui/dom'

import User from '@/components/misc/User.vue'
import {useLabels} from '@/composables/useLabels'
import {useLabelStyles} from '@/composables/useLabelStyles'
import {useProjects} from '@/composables/useProjects'
import {useProjectUserSearch} from '@/composables/useUserSearch'
import {getActiveToken, insertToken, ME_ALIASES, PREFIXES, rankByTitle} from '@/modules/quickAddMagic'
import {useAuthStore} from '@/stores/auth'
import type {Label, User as IUser} from '@/client/generated'

const props = defineProps<{
	input: HTMLInputElement | HTMLTextAreaElement | null,
	projectId: number,
}>()

const model = defineModel<string>({required: true})

interface Row {
	key: string
	// What a pick inserts; empty for rows that only inform.
	value?: string
	text?: string
	hint?: string
	label?: Label
	user?: Pick<IUser, 'name' | 'username'>
	isNewLabel?: boolean
}

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const {labels} = useLabels()
const {getLabelStyles} = useLabelStyles()
const projects = useProjects()

const listId = `quick-add-suggestions-${useId()}`
const list = ref<HTMLElement | null>(null)

// null while the input has no focus.
const caret = ref<number | null>(null)
const token = computed(() => caret.value === null
	? null
	: getActiveToken(model.value, caret.value, PREFIXES[authStore.settings.frontend_settings.quick_add_magic_mode]))
const query = computed(() => token.value?.query ?? '')
const dismissed = ref(false)
const activeIndex = ref(0)

watch(() => [token.value?.kind, token.value?.query], () => {
	activeIndex.value = 0
	dismissed.value = false
})

// shortcut: a `+project` in the text does not change whose members are searched; upgrade if users ask for it.
const searchProjectId = computed(() => props.projectId > 0
	? props.projectId
	: authStore.settings.default_project_id ?? 0)
const debouncedQuery = refDebounced(query, 300)
const userSearch = useProjectUserSearch(searchProjectId, debouncedQuery, () => token.value?.kind === 'assignee')
// keepPrevious data belongs to an older query, so its rows must not be picked.
const usersReady = computed(() => searchProjectId.value <= 0 || (debouncedQuery.value === query.value
	&& !userSearch.isPlaceholderData.value
	&& userSearch.isSuccess.value))

const nothingFound = (): Row => ({key: 'nothing', text: t('task.quickAddSuggest.nothingFound')})

const rows = computed<Row[]>(() => {
	const q = query.value.toLowerCase()
	switch (token.value?.kind) {
		case 'label': {
			const found = rankByTitle(labels.value, q, label => label.title ?? '')
				.map(label => ({key: `l${label.id}`, value: label.title, label}))
			if (found.length > 0) return found
			return q === ''
				? [nothingFound()]
				: [{
					key: 'new',
					value: query.value,
					isNewLabel: true,
					text: t('task.quickAddSuggest.newLabel', {title: query.value}),
				}]
		}
		case 'assignee': {
			const me = authStore.isLinkShareAuth ? null : authStore.info
			const showMe = me && (ME_ALIASES.some(alias => alias.startsWith(q))
				|| [me.username, me.name].some(value => value?.toLowerCase().includes(q)))
			const result: Row[] = showMe
				? [{
					key: 'me',
					value: me.username,
					user: me,
					text: t('task.quickAddSuggest.me', {username: me.username}),
				}]
				: []
			if (!usersReady.value) {
				return [...result, {key: 'loading', text: t('misc.loading')}]
			}
			const members = searchProjectId.value > 0 ? userSearch.users.value : []
			result.push(...members
				.filter(user => user.id !== me?.id)
				.map(user => ({
					key: `u${user.id}`,
					value: user.username,
					user,
					// The User row already shows the username when the name is empty or the same.
					hint: user.name && user.name !== user.username ? user.username : undefined,
				})))
			// Sorting is stable, so Me stays first unless a member's username is exactly the query.
			result.sort((a, b) => Number(b.value?.toLowerCase() === q) - Number(a.value?.toLowerCase() === q))
			return result.length > 0 ? result : [nothingFound()]
		}
		case 'project': {
			const open = projects.projectsArray.filter(project => project.id > 0 && !project.is_archived)
			const found = rankByTitle(open, q, project => project.title)
				.map(project => ({key: `p${project.id}`, value: project.title, text: project.title}))
			return found.length > 0 ? found : [nothingFound()]
		}
		default:
			return []
	}
})

const options = computed(() => rows.value.filter(row => row.value))
const selected = computed(() => options.value[Math.min(activeIndex.value, options.value.length - 1)])
const isOpen = computed(() => token.value !== null && !dismissed.value && rows.value.length > 0)

function updateCaret() {
	caret.value = props.input?.selectionStart ?? null
}

useEventListener(() => props.input, ['input', 'click', 'keyup', 'focus'], updateCaret)
useEventListener(() => props.input, 'blur', () => caret.value = null)

async function pick(value: string) {
	if (!token.value) return
	const result = insertToken(model.value, token.value, value)
	model.value = result.text
	caret.value = result.caret
	await nextTick()
	props.input?.focus()
	props.input?.setSelectionRange(result.caret, result.caret)
}

function move(delta: number) {
	const count = options.value.length
	if (count === 0) return
	activeIndex.value = (options.value.indexOf(selected.value) + delta + count) % count
	nextTick(() => list.value?.querySelector('.is-selected')?.scrollIntoView({block: 'nearest'}))
}

// Capture phase on the input itself, so the host's own Enter/Esc listeners never see a consumed key.
useEventListener(() => props.input, 'keydown', (e: KeyboardEvent) => {
	if (!isOpen.value || e.isComposing) return
	switch (e.key) {
		case 'ArrowDown':
		case 'ArrowUp':
			move(e.key === 'ArrowDown' ? 1 : -1)
			break
		case 'Enter':
			// Only a textarea gets a line break; elsewhere Shift+Enter would create the task.
			if (e.shiftKey && props.input instanceof HTMLTextAreaElement) return
			// Nothing to pick: the host creates the task and resolves the token itself.
			if (!selected.value) return
			pick(selected.value.value!)
			break
		case 'Tab':
			if (e.shiftKey || e.ctrlKey || e.altKey || e.metaKey || !selected.value) return
			pick(selected.value.value!)
			break
		case 'Escape':
			dismissed.value = true
			break
		default:
			return
	}
	e.preventDefault()
	e.stopImmediatePropagation()
}, {capture: true})

watch([list, () => props.input], ([el, input], _, onCleanup) => {
	if (!el || !input) return
	onCleanup(autoUpdate(input, el, () => {
		el.style.inlineSize = `${input.offsetWidth}px`
		computePosition(input, el, {
			strategy: 'fixed',
			placement: 'bottom-start',
			middleware: [offset(4), flip(), shift({padding: 8})],
		}).then(({x, y}) => {
			el.style.left = `${x}px`
			el.style.top = `${y}px`
		})
	}))
}, {flush: 'post'})

watchEffect(onCleanup => {
	const el = props.input
	if (!el) return
	// ARIA in HTML allows the combobox role on a text input but no role on a textarea, which stays a textbox.
	if (el instanceof HTMLInputElement) {
		el.setAttribute('role', 'combobox')
		el.setAttribute('aria-expanded', String(isOpen.value))
	}
	el.setAttribute('aria-autocomplete', 'list')
	if (isOpen.value) {
		el.setAttribute('aria-controls', listId)
		if (selected.value) {
			el.setAttribute('aria-activedescendant', `${listId}-${options.value.indexOf(selected.value)}`)
		}
	}
	onCleanup(() => {
		for (const name of ['role', 'aria-expanded', 'aria-autocomplete', 'aria-controls', 'aria-activedescendant']) {
			el.removeAttribute(name)
		}
	})
})
</script>

<style lang="scss" scoped>
.quick-add-suggestions {
	position: fixed;
	z-index: 4700;
	box-sizing: border-box;
	min-inline-size: 15rem;
	max-inline-size: calc(100vw - 1rem);
	max-block-size: min(50vh, 20rem);
	overflow: hidden auto;
	background: var(--white);
	border: 1px solid var(--primary);
	border-radius: $radius;
	box-shadow: var(--shadow-md);
}

.suggestion {
	display: flex;
	align-items: center;
	gap: .5rem;
	padding: .5rem .75rem;
	color: var(--grey-800);
	cursor: pointer;
	outline: none;

	&.is-selected,
	&:hover:not(.is-inert) {
		background: var(--grey-100);
	}

	&.is-inert {
		color: var(--grey-500);
		cursor: default;
		font-size: .85rem;
	}

	// The row's gap spaces the avatar, so the Me row's text lines up with member names.
	:deep(.avatar-wrapper) {
		margin-inline-end: 0;
	}

	.user {
		gap: .5rem;
	}

	.tag {
		max-inline-size: 100%;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}

.suggestion-text {
	overflow: hidden;
	min-inline-size: 0;
	white-space: nowrap;
	text-overflow: ellipsis;
}

.hint-text {
	margin-inline-start: auto;
	font-size: .75rem;
	color: var(--grey-500);
}

.create-icon {
	color: var(--success);
	font-size: .75rem;
}
</style>
