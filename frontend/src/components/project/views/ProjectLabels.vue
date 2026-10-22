<template>
	<ProjectWrapper
		class="project-labels"
		:is-loading-project="isLoadingProject"
		:project-id="projectId"
		:view-id
	>
		<div
			class="labels-board loader-container"
			:class="{'is-loading': showLoading}"
		>
			<section
				v-for="card in cards"
				:key="card.id"
				class="label-card"
			>
				<header class="label-card__header">
					<span
						class="label-card__dot"
						:style="{backgroundColor: getHexColor(card.label?.hex_color)}"
					/>
					<h2 class="label-card__title">
						{{ card.title }}
					</h2>
					<span class="label-card__count">{{ card.tasks.length }}</span>
				</header>

				<div
					v-if="canWrite"
					class="label-card__add-row"
				>
					<div
						v-if="newTaskCard === card.id"
						class="control"
						:class="{'is-loading': addingTask}"
					>
						<input
							:ref="el => { if (el) newTaskInputEl = el as HTMLInputElement }"
							v-model="newTaskText"
							v-focus.always
							class="input"
							:placeholder="$t('project.kanban.addTaskPlaceholder')"
							type="text"
							@focusout="onNewTaskFocusout"
							@keydown="onNewTaskKeydown($event, card)"
						>
						<QuickAddSuggestions
							v-model="newTaskText"
							:input="newTaskInputEl"
							:project-id="projectId"
						/>
					</div>
					<XButton
						v-else
						class="is-fullwidth label-card__add"
						:shadow="false"
						icon="plus"
						variant="secondary"
						@click="newTaskCard = card.id"
					>
						{{ $t('project.kanban.addTask') }}
					</XButton>
				</div>

				<TaskDraggable
					:model-value="card.tasks"
					:group="{name: 'labels'}"
					:sort="false"
					:disabled="!canWrite"
					:data-label-id="card.id"
					tag="ul"
					:item-key="(task: TaskResponse) => `l${card.id}-t${task.id}`"
					:component-data="{class: ['label-card__tasks', {'dragging-disabled': !canWrite}]}"
					:animation="100"
					:handle="dragHandle"
					:delay-on-touch-only="!isTouchDevice"
					:delay="isTouchDevice ? 300 : 1000"
					ghost-class="task-ghost"
					@end="onDrop"
				>
					<template #item="{element: task}">
						<LabelBoardTask
							:task="task"
							:can-mark-as-done="canWrite"
							:can-edit-status="canWrite && !project?.is_archived"
							:hidden-label-id="card.label?.id"
						>
							<span
								v-if="canWrite"
								class="icon handle"
							>
								<Icon icon="grip-lines" />
							</span>
						</LabelBoardTask>
					</template>
				</TaskDraggable>
			</section>
		</div>
	</ProjectWrapper>
</template>

<script setup lang="ts">
import {computed, ref, toRef} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useI18n} from 'vue-i18n'
import type {SortableEvent} from 'sortablejs'

import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import LabelBoardTask from '@/components/tasks/partials/LabelBoardTask.vue'
import QuickAddSuggestions from '@/components/tasks/partials/QuickAddSuggestions.vue'
import {draggableFor} from '@/components/base/Draggable'

import {allTasksQuery, type TaskResponse} from '@/client/queries/tasks'
import {sortLabelsAlphabetically} from '@/client/queries/labels'
import {useAddTaskLabelMutation, useRemoveTaskLabelMutation} from '@/client/queries/taskMutations'
import type {Label} from '@/client/generated'
import {useCurrentProject} from '@/composables/useCurrentProject'
import {useQuickAddTask} from '@/composables/useQuickAddTask'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {PERMISSIONS as Permissions} from '@/constants/permissions'
import {getHexColor} from '@/helpers/task'
import {useAuthStore} from '@/stores/auth'
import {error} from '@/message'

const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	viewId: number,
}>()

type LabelWithId = Label & Required<Pick<Label, 'id'>>
interface LabelCard {
	id: number
	title: string
	label: LabelWithId | null
	tasks: TaskResponse[]
}

const TaskDraggable = draggableFor<TaskResponse>()
const projectId = toRef(props, 'projectId')

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const {currentProject: project} = useCurrentProject()

const query = useQuery(computed(() => ({
	...allTasksQuery({
		project: projectId.value,
		view: props.viewId,
		params: {
			filter: 'done = false',
			filter_timezone: authStore.settings.timezone,
			// The same expansions as the List view: the status chip needs the buckets.
			expand: ['comment_count', 'is_unread', 'buckets'],
		},
	}),
	// A view filter edited in the settings must show when the board is opened again.
	refetchOnMount: 'always' as const,
})))
const showLoading = useDelayedLoading(query.isLoading)

// Done tasks leave the board at once, before the refetch drops them.
// Newest first, like Google Tasks: a task added from the top of a card shows up right there.
const tasks = computed(() => (query.data.value ?? [])
	.filter(task => !task.done)
	.sort((a, b) => (b.id ?? 0) - (a.id ?? 0)))

const cards = computed<LabelCard[]>(() => {
	// Phones snap the board to a card: a lone "No label" card shown before the load would stay snapped
	// while the label cards appear in front of it, opening the board at its far end.
	if (!query.data.value) return []
	const used = new Map<number, LabelWithId>()
	for (const task of tasks.value) {
		for (const label of task.labels) {
			if (label.id) used.set(label.id, label as LabelWithId)
		}
	}
	// Spec 4.5 sorts by the ru locale whatever the UI language: it puts Cyrillic titles before Latin ones.
	const labelCards = (sortLabelsAlphabetically([...used.values()], 'ru-RU') as LabelWithId[]).map(label => ({
		id: label.id,
		title: label.title ?? '',
		label,
		tasks: tasks.value.filter(task => task.labels.some(item => item.id === label.id)),
	}))
	return [...labelCards, {
		id: 0,
		title: t('project.labels.noLabel'),
		label: null,
		tasks: tasks.value.filter(task => task.labels.length === 0),
	}]
})

const canWrite = computed(() =>
	typeof project.value?.max_permission === 'number' &&
	project.value.max_permission > Permissions.READ &&
	project.value.id > 0,
)

const isTouchDevice = !window.matchMedia('(hover: hover) and (pointer: fine)').matches
const dragHandle = isTouchDevice ? '.handle' : undefined

const addLabel = useAddTaskLabelMutation()
const removeLabel = useRemoveTaskLabelMutation()

// shortcut: a refetch cancelled by a newer one is swallowed (the newer one carries the data),
// and two quick drops may flicker briefly; add optimistic moves if that ever bothers anyone.
const refetchBoard = () => query.refetch().catch(() => {})

// A second drop of the same task during its relabel would remove both labels, so it snaps back instead.
const relabeling = new Set<number>()

async function onDrop(e: SortableEvent) {
	const from = cards.value.find(card => card.id === Number(e.from.dataset.labelId))
	const to = cards.value.find(card => card.id === Number(e.to.dataset.labelId))
	const task = tasks.value.find(item => item.id === Number(e.item.dataset.taskId))
	if (!from || !to || !task || from === to || relabeling.has(task.id)) return
	relabeling.add(task.id)
	try {
		if (to.label && !task.labels.some(label => label.id === to.id)) {
			await addLabel.mutateAsync({taskId: task.id, label: to.label})
		}
		if (from.label) {
			await removeLabel.mutateAsync({taskId: task.id, label: from.label})
		}
	} catch { /* The mutation reports the error; vuedraggable has already put the row back. */ } finally {
		relabeling.delete(task.id)
		// The relabel can move the task in or out of a filtered view.
		await refetchBoard()
	}
}

const {createNewTask} = useQuickAddTask()
const newTaskCard = ref<number | null>(null)
const newTaskText = ref('')
const newTaskInputEl = ref<HTMLInputElement | null>(null)
const addingTask = ref(false)

function onNewTaskFocusout(e: FocusEvent) {
	// A tap on a suggestion must not close the input (backup for the list's own pointerdown guard).
	if ((e.relatedTarget as Element | null)?.closest('.quick-add-suggestions')) return
	newTaskCard.value = null
}

function onNewTaskKeydown(e: KeyboardEvent, card: LabelCard) {
	if (e.isComposing) return
	if (e.key === 'Enter') {
		if (e.repeat) return
		e.preventDefault()
		addTask(card)
	} else if (e.key === 'Escape') {
		(e.target as HTMLInputElement).blur()
	}
}

async function addTask(card: LabelCard) {
	const title = newTaskText.value
	if (title.trim() === '' || addingTask.value) return
	addingTask.value = true
	try {
		const task = await createNewTask({title, project_id: projectId.value})
		if (newTaskText.value === title) newTaskText.value = ''
		if (card.label && task.id && !task.labels?.some(label => label.id === card.id)) {
			try {
				await addLabel.mutateAsync({taskId: task.id, label: card.label})
			} catch {
				// The task exists by now, so resubmitting the text would only create a duplicate.
				error({message: t('project.labels.labelNotApplied', {label: card.title})})
			}
		}
		// The create's own refetch ran before the label existed, so a label-filtered view would miss the task.
		await refetchBoard()
	} finally {
		addingTask.value = false
	}
}
</script>

<style lang="scss" scoped>
// The Kanban board height, so each card scrolls on its own inside the page.
$board-height: '100vh - 4.5rem - 1.5rem - 1rem - 1.5rem - 11px';

.labels-board {
	display: flex;
	align-items: flex-start;
	gap: 1rem;
	overflow: auto hidden;
	block-size: calc(#{$board-height});
	margin: 0 -1.5rem;
	padding: .25rem 1.5rem 1rem;

	@media screen and (max-width: $tablet) {
		block-size: calc(#{$board-height} + #{$switch-view-height} - 1rem + 9px);
		margin: 0 -.5rem;
		padding-inline: .5rem;
		scroll-snap-type: x mandatory;
		scroll-padding-inline: .5rem;
	}
}

.label-card {
	// Contains the rows' absolutely positioned screen-reader inputs, which would otherwise widen the page.
	position: relative;
	flex: 0 0 340px;
	max-inline-size: calc(100vw - 2rem);
	max-block-size: 100%;
	display: flex;
	flex-direction: column;
	background-color: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: .75rem;
	box-shadow: var(--shadow-sm);
	overflow: hidden;

	@media screen and (max-width: $tablet) {
		scroll-snap-align: start;
	}
}

.label-card__header {
	display: flex;
	align-items: center;
	gap: .625rem;
	padding: 1rem 1rem .5rem;
}

.label-card__dot {
	flex-shrink: 0;
	inline-size: .75rem;
	block-size: .75rem;
	border-radius: 50%;
	background-color: var(--grey-300);
}

.label-card__title {
	flex: 1;
	min-inline-size: 0;
	margin: 0;
	overflow: hidden;
	color: var(--text-strong);
	font-size: 1rem;
	font-weight: 600;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.label-card__count {
	color: var(--grey-500);
	font-size: .875rem;
	font-variant-numeric: tabular-nums;
}

.label-card__tasks {
	flex: 1 1 auto;
	// An empty card still needs room to drop onto.
	min-block-size: 2.5rem;
	margin: 0;
	// The bottom padding absorbs the pixel the row icons poke out, which would otherwise show a scrollbar.
	padding: 0 .5rem .25rem;
	overflow: hidden auto;
	list-style: none;
}

.label-card__add-row {
	padding: 0 .5rem .25rem;

	.label-card__add {
		justify-content: flex-start;
		background-color: transparent;
		color: var(--primary);
		font-weight: 600;
		text-transform: none;

		&:hover {
			background-color: var(--grey-100);
		}
	}
}

.task-ghost {
	border-radius: $radius;
	background: var(--grey-100);
	border: 2px dashed var(--grey-300);

	* {
		opacity: 0;
	}
}

:deep(.board-task .handle) {
	cursor: grab;
	margin-inline-end: .25rem;
	color: var(--grey-400);
}

@media (hover: hover) and (pointer: fine) {
	:deep(.board-task .handle) {
		display: none;
	}
}

:deep(.label-card__tasks:not(.dragging-disabled) .board-task) {
	cursor: grab;
	-webkit-touch-callout: none;
	user-select: none;
	touch-action: manipulation;

	&:active {
		cursor: grabbing;
	}
}
</style>
