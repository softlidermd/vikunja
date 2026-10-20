<template>
	<ProjectWrapper
		class="project-table"
		:is-loading-project="isLoadingProject"
		:project-id="projectId"
		:view-id
	>
		<template #header>
			<div class="filter-container">
				<Popup
					placement="bottom-start"
					:anchor="columnsTriggerEl"
				>
					<template #trigger="{toggle}">
						<XButton
							ref="columnsTrigger"
							icon="th"
							variant="secondary"
							class="mie-2"
							@click.prevent.stop="toggle()"
						>
							{{ $t('project.table.columns') }}
						</XButton>
					</template>
					<template #content="{isOpen}">
						<Card
							class="columns-filter"
							:class="{'is-open': isOpen}"
						>
							<FancyCheckbox v-model="activeColumns.index">
								#
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.done">
								{{ $t('task.attributes.done') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.project">
								{{ $t('task.attributes.project') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.title">
								{{ $t('task.attributes.title') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.priority">
								{{ $t('task.attributes.priority') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.labels">
								{{ $t('task.attributes.labels') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.assignees">
								{{ $t('task.attributes.assignees') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.commentCount">
								{{ $t('task.attributes.commentCount') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.dueDate">
								{{ $t('task.attributes.dueDate') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.startDate">
								{{ $t('task.attributes.startDate') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.endDate">
								{{ $t('task.attributes.endDate') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.percentDone">
								{{ $t('task.attributes.percentDone') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.doneAt">
								{{ $t('task.attributes.doneAt') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.created">
								{{ $t('task.attributes.created') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.updated">
								{{ $t('task.attributes.updated') }}
							</FancyCheckbox>
							<FancyCheckbox v-model="activeColumns.createdBy">
								{{ $t('task.attributes.createdBy') }}
							</FancyCheckbox>
						</Card>
					</template>
				</Popup>
				<FilterPopup
					v-if="!isSavedFilterProject({id: projectId})"
					v-model="params"
					:view-id="viewId"
					:project-id="projectId"
					@update:modelValue="taskList.loadTasks()"
				/>
			</div>
		</template>

		<template #default>
			<div
				:class="{'is-loading': loading}"
				class="loader-container"
			>
				<Card
					:padding="false"
					:has-content="false"
				>
					<AddTask
						v-if="canWrite"
						class="table-view__add-task d-print-none"
					/>
					<div class="has-horizontal-overflow">
						<table class="table has-actions is-hoverable is-fullwidth mbe-0">
							<thead>
								<tr>
									<th
										v-if="activeColumns.index"
										:aria-sort="ariaSort('index')"
									>
										#
										<Sort
											:order="sortBy.index"
											label="#"
											@click="sort('index', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.done"
										v-tooltip="$t('task.attributes.done')"
										:aria-sort="ariaSort('done')"
									>
										<!-- An icon instead of the word: the column only holds a checkbox. -->
										<Icon icon="check" />
										<span class="is-sr-only">{{ $t('task.attributes.done') }}</span>
										<Sort
											:order="sortBy.done"
											:label="$t('task.attributes.done')"
											@click="sort('done', $event)"
										/>
									</th>
									<th v-if="activeColumns.project">
										{{ $t('task.attributes.project') }}
									</th>
									<th
										v-if="activeColumns.title"
										:aria-sort="ariaSort('title')"
									>
										{{ $t('task.attributes.title') }}
										<Sort
											:order="sortBy.title"
											:label="$t('task.attributes.title')"
											@click="sort('title', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.priority"
										:aria-sort="ariaSort('priority')"
									>
										{{ $t('task.attributes.priority') }}
										<Sort
											:order="sortBy.priority"
											:label="$t('task.attributes.priority')"
											@click="sort('priority', $event)"
										/>
									</th>
									<th v-if="activeColumns.labels">
										{{ $t('task.attributes.labels') }}
									</th>
									<th v-if="activeColumns.assignees">
										{{ $t('task.attributes.assignees') }}
									</th>
									<th
										v-if="activeColumns.dueDate"
										:aria-sort="ariaSort('due_date')"
									>
										{{ $t('task.attributes.dueDate') }}
										<Sort
											:order="sortBy.due_date"
											:label="$t('task.attributes.dueDate')"
											@click="sort('due_date', $event)"
										/>
									</th>
									<th v-if="activeColumns.commentCount">
										{{ $t('task.attributes.commentCount') }}
									</th>
									<th
										v-if="activeColumns.startDate"
										:aria-sort="ariaSort('start_date')"
									>
										{{ $t('task.attributes.startDate') }}
										<Sort
											:order="sortBy.start_date"
											:label="$t('task.attributes.startDate')"
											@click="sort('start_date', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.endDate"
										:aria-sort="ariaSort('end_date')"
									>
										{{ $t('task.attributes.endDate') }}
										<Sort
											:order="sortBy.end_date"
											:label="$t('task.attributes.endDate')"
											@click="sort('end_date', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.percentDone"
										:aria-sort="ariaSort('percent_done')"
									>
										{{ $t('task.attributes.percentDone') }}
										<Sort
											:order="sortBy.percent_done"
											:label="$t('task.attributes.percentDone')"
											@click="sort('percent_done', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.doneAt"
										:aria-sort="ariaSort('done_at')"
									>
										{{ $t('task.attributes.doneAt') }}
										<Sort
											:order="sortBy.done_at"
											:label="$t('task.attributes.doneAt')"
											@click="sort('done_at', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.created"
										:aria-sort="ariaSort('created')"
									>
										{{ $t('task.attributes.created') }}
										<Sort
											:order="sortBy.created"
											:label="$t('task.attributes.created')"
											@click="sort('created', $event)"
										/>
									</th>
									<th
										v-if="activeColumns.updated"
										:aria-sort="ariaSort('updated')"
									>
										{{ $t('task.attributes.updated') }}
										<Sort
											:order="sortBy.updated"
											:label="$t('task.attributes.updated')"
											@click="sort('updated', $event)"
										/>
									</th>
									<th v-if="activeColumns.createdBy">
										{{ $t('task.attributes.createdBy') }}
									</th>
								</tr>
							</thead>
							<tbody>
								<tr
									v-for="t in tasks"
									:key="t.id"
								>
									<td v-if="activeColumns.index">
										<RouterLink :to="taskDetailRoutes[t.id]">
											{{ getTaskIdentifier(t) }}
										</RouterLink>
									</td>
									<td v-if="activeColumns.done">
										<FancyCheckbox
											v-if="canWrite"
											:model-value="t.done"
											:aria-label="$t('task.detail.markAsDone', {task: t.title})"
											@update:modelValue="v => saveTask(t, {done: v})"
										/>
										<template v-else-if="t.done">
											<Icon icon="check" />
											<span class="is-sr-only">{{ $t('task.attributes.done') }}</span>
										</template>
									</td>
									<td v-if="activeColumns.project">
										<RouterLink
											v-if="projectList.projects[t.project_id]"
											:to="{ name: 'project.index', params: { projectId: t.project_id } }"
										>
											{{ projectList.projects[t.project_id].title }}
										</RouterLink>
									</td>
									<td v-if="activeColumns.title">
										<TaskGlanceTooltip :task="t">
											<RouterLink :to="taskDetailRoutes[t.id]">
												{{ t.title }}
											</RouterLink>
										</TaskGlanceTooltip>
									</td>
									<td v-if="activeColumns.priority">
										<PrioritySelect
											v-if="canWrite"
											class="is-small"
											:model-value="t.priority"
											@update:modelValue="p => saveTask(t, {priority: p})"
										/>
										<PriorityLabel
											v-else
											:priority="t.priority"
											:show-all="true"
										/>
									</td>
									<td v-if="activeColumns.labels">
										<Labels :labels="t.labels" />
									</td>
									<td v-if="activeColumns.assignees">
										<BaseButton
											v-if="canEditAssignees"
											:aria-label="assigneesLabel($t('task.attributes.assignees'), t.assignees)"
											@click="openAssignees(t, $event)"
										>
											<span
												v-if="t.assignees.length > 0"
												class="table-assignees"
											>
												<User
													v-for="a in t.assignees"
													:key="a.id"
													:user="a"
													:avatar-size="20"
													is-inline
												/>
											</span>
											<Icon
												v-else
												icon="plus"
											/>
										</BaseButton>
										<span
											v-else-if="t.assignees.length > 0"
											class="table-assignees"
										>
											<User
												v-for="a in t.assignees"
												:key="a.id"
												:user="a"
												:avatar-size="20"
												is-inline
											/>
										</span>
									</td>
									<template v-if="activeColumns.dueDate">
										<td
											v-if="canWrite"
											class="due-date-cell"
										>
											<Datepicker
												:model-value="dueValue(t)"
												:choose-date-label="$t('task.detail.chooseDueDate')"
												empty-label="-"
												@update:modelValue="d => dueDraft[t.id] = d"
												@closeOnChange="saveDue(t)"
											/>
										</td>
										<DateTableCell
											v-else
											:date="t.due_date"
										/>
									</template>
									<td v-if="activeColumns.commentCount">
										<CommentCount :task="t" />
									</td>
									<DateTableCell
										v-if="activeColumns.startDate"
										:date="t.start_date"
									/>
									<DateTableCell
										v-if="activeColumns.endDate"
										:date="t.end_date"
									/>
									<td v-if="activeColumns.percentDone">
										{{ t.percent_done * 100 }}%
									</td>
									<DateTableCell
										v-if="activeColumns.doneAt"
										:date="t.done_at"
									/>
									<DateTableCell
										v-if="activeColumns.created"
										:date="t.created"
									/>
									<DateTableCell
										v-if="activeColumns.updated"
										:date="t.updated"
									/>
									<td v-if="activeColumns.createdBy">
										<User
											:avatar-size="27"
											:show-username="false"
											:user="t.created_by ?? {}"
										/>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
					<div
						v-if="assigneeTask"
						class="table-assignees-popup"
					>
						<Popup
							v-model:open="assigneesOpen"
							:anchor="assigneeAnchor"
							placement="bottom-start"
							sheet-on-mobile
							:sheet-title="$t('task.attributes.assignees')"
						>
							<template #content>
								<Card class="table-assignees-card">
									<EditAssignees
										:model-value="assigneeTask.assignees"
										:task-id="assigneeTask.id"
										:project-id="assigneeTask.project_id"
									/>
								</Card>
							</template>
						</Popup>
					</div>

					<Pagination
						:total-pages="totalPages"
						:current-page="currentPage"
					/>
				</Card>
			</div>
		</template>
	</ProjectWrapper>
</template>

<script setup lang="ts">
import {computed, ref, type ComponentPublicInstance, type Ref, watch} from 'vue'

import {useStorage} from '@vueuse/core'

import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import User from '@/components/misc/User.vue'
import PriorityLabel from '@/components/tasks/partials/PriorityLabel.vue'
import Labels from '@/components/tasks/partials/Labels.vue'
import TaskGlanceTooltip from '@/components/tasks/partials/TaskGlanceTooltip.vue'
import DateTableCell from '@/components/tasks/partials/DateTableCell.vue'
import CommentCount from '@/components/tasks/partials/CommentCount.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import Sort from '@/components/tasks/partials/Sort.vue'
import FilterPopup from '@/components/project/partials/FilterPopup.vue'
import Pagination from '@/components/misc/Pagination.vue'
import Popup from '@/components/misc/Popup.vue'
import PrioritySelect from '@/components/tasks/partials/PrioritySelect.vue'
import EditAssignees from '@/components/tasks/partials/EditAssignees.vue'
import Datepicker from '@/components/input/Datepicker.vue'
import BaseButton from '@/components/base/BaseButton.vue'

import type {SortBy} from '@/composables/useTaskList'
import {useTaskList} from '@/composables/useTaskList'
import type {TaskResponse} from '@/client/queries/tasks'
import {useTableSort} from '@/composables/useTableSort'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import AddTask from '@/components/tasks/AddTask.vue'
import {useCurrentProject} from '@/composables/useCurrentProject'
import {PERMISSIONS as Permissions} from '@/constants/permissions'
import {getTaskIdentifier} from '@/helpers/task'
import { camelCase } from 'change-case'
import {isSavedFilterProject} from '@/client/queries/projects'
import {useProjects} from '@/composables/useProjects'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'
import type {Task} from '@/client/generated'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {useAuthStore} from '@/stores/auth'
import {captureClientRequestContext, isClientRequestContextCurrent} from '@/client/requestContext'
import {getDisplayName} from '@/helpers/user'

const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	viewId: number,
}>()

const projectList = useProjects()
const {currentProject: project} = useCurrentProject()

const canWrite = computed(() => !project.value?.is_archived &&
	typeof project.value?.max_permission === 'number' &&
	project.value.max_permission > Permissions.READ &&
	project.value.id > 0)
const authStore = useAuthStore()
// Link shares may not search users, so they cannot pick assignees.
const canEditAssignees = computed(() => canWrite.value && !authStore.isLinkShareAuth)

const columnsTrigger = ref<ComponentPublicInstance | null>(null)
const columnsTriggerEl = computed<HTMLElement | null>(() => (columnsTrigger.value?.$el as HTMLElement) ?? null)

const ACTIVE_COLUMNS_DEFAULT = {
	index: true,
	done: true,
	project: false,
	title: true,
	priority: false,
	labels: true,
	assignees: true,
	dueDate: true,
	startDate: false,
	endDate: false,
	percentDone: false,
	created: false,
	updated: false,
	createdBy: false,
	doneAt: false,
	commentCount: false,
}

const SORT_BY_DEFAULT: SortBy = {
	index: 'desc',
}

const activeColumns = useStorage('tableViewColumns', {...ACTIVE_COLUMNS_DEFAULT})
const sortBy = useStorage<SortBy>('tableViewSortBy', {...SORT_BY_DEFAULT})

const taskList = useTaskList(
	() => props.projectId, 
	() => props.viewId, 
	sortBy.value,
	() => ['comment_count', 'is_unread'],
)

const {
	params,
	totalPages,
	currentPage,
	sortByParam,
} = taskList
const loading = useDelayedLoading(taskList.loading)
const tasks: Ref<TaskResponse[]> = taskList.tasks

watch(
	() => activeColumns.value,
	() => setActiveColumnsSortParam(),
	{deep: true},
)

const {sort, ariaSort} = useTableSort<keyof SortBy>(sortBy, setActiveColumnsSortParam)

function setActiveColumnsSortParam() {
	sortByParam.value = (Object.keys(sortBy.value) as (keyof SortBy)[])
		.filter(prop => activeColumns.value[camelCase(prop) as keyof typeof activeColumns.value])
		.reduce<SortBy>((obj, key) => {
			obj[key] = sortBy.value[key]
			return obj
		}, {})
}

const updateTask = useUpdateTaskMutation(true)

// Saves of one task run one after another: each response carries the whole task,
// so a late response to an earlier edit would put the old value of a newer edit back.
const pendingSaves = new Map<number, Promise<void>>()

// Partial patch, so concurrent edits to other fields are not overwritten.
function saveTask(task: TaskResponse, patch: Partial<Task>) {
	// The mutation captures its context only when it starts, so fence a queued save to the session that made the edit.
	const request = captureClientRequestContext()
	const save = (pendingSaves.get(task.id) ?? Promise.resolve()).then(async () => {
		if (!isClientRequestContextCurrent(request)) {
			return
		}
		try {
			await updateTask.mutateAsync({
				id: task.id,
				...patch,
			})
		} catch {
			// rollback and error toast are done by contextMutationOptions
		}
	})
	pendingSaves.set(task.id, save)
	save.then(() => {
		if (pendingSaves.get(task.id) === save) {
			pendingSaves.delete(task.id)
		}
	})
	return save
}

const dueDraft = ref<Record<number, Date | null>>({})

function dueValue(task: TaskResponse) {
	if (task.id in dueDraft.value) {
		return dueDraft.value[task.id] ?? null
	}
	return parseDateOrNull(task.due_date)
}

async function saveDue(task: TaskResponse) {
	if (!(task.id in dueDraft.value)) {
		return
	}
	const draft = dueDraft.value[task.id]
	// '' clears the date (taskWriteBody maps it to the api's zero date).
	await saveTask(task, {due_date: draft?.toISOString() ?? ''})
	// Keep the draft until the save lands, so the old date doesn't flash back; a newer pick stays.
	if (dueDraft.value[task.id] === draft) {
		delete dueDraft.value[task.id]
	}
}

const assigneesOpen = ref(false)
const assigneeTaskId = ref<number | null>(null)
const assigneeAnchor = ref<HTMLElement | null>(null)
const assigneeTask = computed(() => tasks.value.find(x => x.id === assigneeTaskId.value))
// The task left the page (paging, filter): close, so the popup doesn't reopen on a detached anchor.
watch(assigneeTask, task => {
	if (!task) {
		assigneesOpen.value = false
		assigneeAnchor.value = null
	}
})

// The aria-label replaces the button content, so it has to name the people too.
function assigneesLabel(label: string, assignees: TaskResponse['assignees']) {
	return assignees.length > 0
		? `${label}: ${assignees.map(getDisplayName).join(', ')}`
		: label
}

function openAssignees(task: TaskResponse, event: MouseEvent) {
	assigneeTaskId.value = task.id
	assigneeAnchor.value = event.currentTarget as HTMLElement
	assigneesOpen.value = true
}

// TODO: re-enable opening task detail in modal
// const router = useRouter()
const taskDetailRoutes = computed(() => Object.fromEntries(
	tasks.value.map(({id}) => ([
		id,
		{
			name: 'task.detail',
			params: {id},
			// state: { backdropView: router.currentRoute.value.fullPath },
		},
	])),
))
</script>

<style lang="scss" scoped>
.table-view__add-task {
	padding: 1rem;
}

.table {
	background: transparent;
	overflow-x: auto;
	overflow-y: hidden;

	th {
		white-space: nowrap;
	}

	// The relative date ("через 19 часов") would otherwise wrap into a tall row.
	.due-date-cell {
		white-space: nowrap;
	}

	.user {
		margin: 0;
	}

	// One assignee per line, avatar plus name, so it is clear who it is without hovering.
	.table-assignees {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: .25rem;
		white-space: nowrap;
		color: var(--text);
	}
}

.columns-filter {
	margin: 0;

	:deep(.card-content .content) {
		display: flex;
		flex-direction: column;
	}

}

// Popup scrolls (overflow-y: auto), which would clip the absolutely positioned assignee search results.
.table-assignees-popup {
	:deep(.popup) {
		overflow: visible;
	}

	// The popover shrinks to its content, which would squeeze the search input.
	.card {
		inline-size: 20rem;
		max-inline-size: calc(100vw - 2rem);
		margin: 0;
	}
}

// The mobile sheet sits at the screen bottom and sizes to its content, so absolute results would hang off-screen.
.bottom-sheet .table-assignees-card :deep(.search-results) {
	position: static;
}

.link-share-view .card {
	border: none;
	box-shadow: none;
}
</style>
