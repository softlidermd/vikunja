<template>
	<div
		ref="taskViewContainer"
		class="loader-container task-view-container"
		:class="{
			'is-loading': taskBusy || !visible,
			'is-modal': isModal,
		}"
	>
		<!-- Removing everything until the task is loaded to prevent empty initialization of other components -->
		<div
			v-if="visible"
			class="task-view"
		>
			<BaseButton
				v-if="!isModal"
				class="back-button mbs-2"
				@click="lastProject ? router.back() : router.push(projectRoute)"
			>
				<Icon icon="arrow-left" />
				{{ $t('task.detail.back') }}
			</BaseButton>
			<div class="task-topbar">
				<nav
					v-if="project?.id"
					aria-label="Breadcrumb"
					class="subtitle"
				>
					<template
						v-for="p in projectList.getAncestors(project)"
						:key="p.id"
					>
						<a
							v-if="typeof router.options.history.state.back === 'string' && router.options.history.state.back.includes('/projects/'+p.id+'/')"
							v-shortcut="p.id === project?.id ? SHORTCUTS.taskDetail.openProject : ''"
							@click="router.back()"
						>
							{{ getProjectTitle(p) }}
						</a>
						<RouterLink
							v-else
							v-shortcut="p.id === project?.id ? SHORTCUTS.taskDetail.openProject : ''"
							:to="{ name: 'project.index', params: { projectId: p.id } }"
						>
							{{ getProjectTitle(p) }}
						</RouterLink>
						<span
							v-if="p.id !== project?.id"
							class="has-text-grey-light"
						> &gt; </span>
					</template>
					<BucketSelect
						:task="task"
						:can-write="canWrite"
					/>
				</nav>
				<div
					v-if="canWrite"
					class="task-topbar__actions d-print-none"
				>
					<BaseButton
						v-shortcut="SHORTCUTS.taskDetail.favorite"
						v-tooltip="task.is_favorite ? $t('task.detail.actions.unfavorite') : $t('task.detail.actions.favorite')"
						class="topbar-button"
						:class="{'is-favorite': task.is_favorite}"
						:aria-pressed="task.is_favorite"
						:aria-label="$t('task.detail.actions.favorite')"
						@click="toggleFavorite"
					>
						<Icon :icon="task.is_favorite ? 'star' : ['far', 'star']" />
					</BaseButton>
					<Dropdown
						v-cy="'taskDetail.more'"
						eager
						class="more-menu"
						:trigger-label="$t('task.detail.moreActions')"
					>
						<template #default="{close}">
							<TaskSubscription
								type="dropdown"
								entity="task"
								:entity-id="task.id"
								:model-value="task.subscription ?? null"
								@toggle="close(); toggleSubscription($event)"
							/>
							<DropdownItem
								v-shortcut="SHORTCUTS.taskDetail.attachments"
								icon="paperclip"
								@click="close(); openAttachments()"
							>
								{{ $t('task.detail.actions.attachments') }}
							</DropdownItem>
							<DropdownItem
								v-shortcut="SHORTCUTS.taskDetail.relatedTasks"
								icon="sitemap"
								@click="close(); setRelatedTasksActive()"
							>
								{{ $t('task.detail.actions.relatedTasks') }}
							</DropdownItem>
							<DropdownItem
								v-if="timeTrackingEnabled"
								v-cy="'taskTrackTimeAction'"
								:icon="['far', 'clock']"
								@click="close(); setFieldActive('timeTracking')"
							>
								{{ $t('task.detail.actions.timeTracking') }}
							</DropdownItem>
							<hr class="dropdown-divider">
							<DropdownItem
								v-shortcut="SHORTCUTS.taskDetail.moveProject"
								icon="list"
								@click="close(); setFieldActive('moveProject')"
							>
								{{ $t('task.detail.actions.moveProject') }}
							</DropdownItem>
							<DropdownItem
								icon="copy"
								@click="close(); duplicateCurrentTask()"
							>
								{{ $t('task.detail.actions.duplicate') }}
							</DropdownItem>
							<hr class="dropdown-divider">
							<DropdownItem
								v-shortcut="SHORTCUTS.taskDetail.delete"
								icon="trash-alt"
								class="has-text-danger"
								@click="close(); showDeleteModal = true"
							>
								{{ $t('task.detail.actions.delete') }}
							</DropdownItem>
						</template>
					</Dropdown>
				</div>
			</div>

			<div class="task-title-row">
				<BaseButton
					v-if="canWrite"
					v-shortcut="SHORTCUTS.taskDetail.done"
					v-cy="'taskDetail.done'"
					v-tooltip="task.done ? $t('task.detail.undone') : $t('task.detail.done')"
					class="done-toggle d-print-none"
					:class="{'done-toggle--done': task.done}"
					:aria-pressed="task.done"
					:aria-label="$t('task.detail.done')"
					@click="toggleTaskDone()"
				>
					<Icon icon="check" />
				</BaseButton>
				<Heading
					ref="heading"
					:task="task"
					:can-write="canWrite"
				>
					<CreatedUpdated :task="task" />
				</Heading>
			</div>

			<div class="properties">
				<div
					v-if="activeFields.assignees"
					class="property assignees"
				>
					<!-- Assignees -->
					<div class="detail-title">
						<Icon icon="users" />
						{{ $t('task.attributes.assignees') }}
					</div>
					<EditAssignees
						v-if="canWrite"
						:ref="e => setFieldRef('assignees', e)"
						v-model="task.assignees"
						:project-id="task.project_id"
						:task-id="task.id"
					/>
					<AssigneeList
						v-else
						:assignees="task.assignees ?? []"
						class="mbs-2"
					/>
				</div>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.priority"
						class="property"
					>
						<!-- Priority -->
						<div class="detail-title">
							<Icon icon="exclamation-circle" />
							{{ $t('task.attributes.priority') }}
						</div>
						<PrioritySelect
							:ref="e => setFieldRef('priority', e)"
							v-model="task.priority"
							:disabled="!canWrite"
							@update:modelValue="setPriority"
						/>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.size"
						class="property"
					>
						<div class="detail-title">
							<Icon icon="ruler-horizontal" />
							{{ $t('task.attributes.size') }}
						</div>
						<SizeSelect
							:ref="e => setFieldRef('size', e)"
							v-model="task.size"
							:disabled="!canWrite"
							@update:modelValue="setSize"
						/>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.dueDate"
						class="property"
					>
						<!-- Due Date -->
						<div class="detail-title">
							<Icon icon="calendar" />
							{{ $t('task.attributes.dueDate') }}
						</div>
						<div class="date-input">
							<Datepicker
								ref="dueDatePicker"
								v-model="dueDateInput"
								:choose-date-label="$t('task.detail.chooseDueDate')"
								:title="$t('task.attributes.dueDate')"
								:disabled="taskBusy || !canWrite"
								@closeOnChange="saveTask()"
							/>
							<BaseButton
								v-if="task.due_date && canWrite"
								class="remove"
								:aria-label="$t('task.detail.removeDueDate')"
								@click="() => {task.due_date = '';saveTask()}"
							>
								<span class="icon is-small">
									<Icon icon="times" />
								</span>
							</BaseButton>
						</div>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.percentDone"
						class="property"
					>
						<!-- Progress -->
						<div class="detail-title">
							<Icon icon="percent" />
							{{ $t('task.attributes.percentDone') }}
						</div>
						<PercentDoneSelect
							:ref="e => setFieldRef('percentDone', e)"
							v-model="task.percent_done"
							:disabled="!canWrite"
							@update:modelValue="setPercentDone"
						/>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.startDate"
						class="property"
					>
						<!-- Start Date -->
						<div class="detail-title">
							<Icon icon="play" />
							{{ $t('task.attributes.startDate') }}
						</div>
						<div class="date-input">
							<Datepicker
								ref="startDatePicker"
								v-model="startDateInput"
								:choose-date-label="$t('task.detail.chooseStartDate')"
								:title="$t('task.attributes.startDate')"
								:disabled="taskBusy || !canWrite"
								@closeOnChange="saveTask()"
							/>
							<BaseButton
								v-if="task.start_date && canWrite"
								class="remove"
								:aria-label="$t('task.detail.removeStartDate')"
								@click="() => {task.start_date = '';saveTask()}"
							>
								<span class="icon is-small">
									<Icon icon="times" />
								</span>
							</BaseButton>
						</div>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.endDate"
						class="property"
					>
						<!-- End Date -->
						<div class="detail-title">
							<Icon icon="stop" />
							{{ $t('task.attributes.endDate') }}
						</div>
						<div class="date-input">
							<Datepicker
								ref="endDatePicker"
								v-model="endDateInput"
								:choose-date-label="$t('task.detail.chooseEndDate')"
								:title="$t('task.attributes.endDate')"
								:disabled="taskBusy || !canWrite"
								@closeOnChange="saveTask()"
							/>
							<BaseButton
								v-if="task.end_date && canWrite"
								class="remove"
								:aria-label="$t('task.detail.removeEndDate')"
								@click="() => {task.end_date = '';saveTask()}"
							>
								<span class="icon is-small">
									<Icon icon="times" />
								</span>
							</BaseButton>
						</div>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.reminders"
						class="property"
					>
						<!-- Reminders -->
						<div class="detail-title">
							<Icon :icon="['far', 'clock']" />
							{{ $t('task.attributes.reminders') }}
						</div>
						<Reminders
							:ref="e => setFieldRef('reminders', e)"
							v-model="task.reminders"
							:default-relative-to="remindersDefaultRelativeTo"
							:disabled="!canWrite"
							@update:modelValue="saveTask()"
						/>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.repeatAfter"
						class="property"
					>
						<!-- Repeat after -->
						<div class="is-flex is-justify-content-space-between">
							<div class="detail-title">
								<Icon icon="history" />
								{{ $t('task.attributes.repeat') }}
							</div>
							<BaseButton
								v-if="canWrite"
								class="remove"
								:aria-label="$t('task.detail.removeRepeat')"
								@click="removeRepeatAfter"
							>
								<span class="icon is-small">
									<Icon icon="times" />
								</span>
							</BaseButton>
						</div>
						<RepeatAfter
							:ref="e => setFieldRef('repeatAfter', e)"
							:model-value="task"
							:disabled="!canWrite"
							@update:modelValue="saveTask($event)"
						/>
					</div>
				</CustomTransition>
				<CustomTransition
					name="flash-background"
					appear
				>
					<div
						v-if="activeFields.color"
						class="property"
					>
						<!-- Color -->
						<div class="detail-title">
							<Icon icon="fill-drip" />
							{{ $t('task.attributes.color') }}
						</div>
						<ColorPicker
							:ref="e => setFieldRef('color', e)"
							v-model="taskColor"
							menu-position="bottom"
							@update:modelValue="saveTask()"
						/>
					</div>
				</CustomTransition>

				<!-- Labels -->
				<div
					v-if="activeFields.labels"
					class="property labels-list"
				>
					<div class="detail-title">
						<span class="icon is-grey">
							<Icon icon="tags" />
						</span>
						{{ $t('task.attributes.labels') }}
					</div>
					<EditLabels
						:ref="e => setFieldRef('labels', e)"
						v-model="task.labels"
						:project-id="task.project_id"
						:disabled="!canWrite"
						:task-id="taskId"
						:creatable="!authStore.isLinkShareAuth"
						:creation-disabled-message="authStore.isLinkShareAuth ? $t('task.label.linkShareCannotCreate') : ''"
					/>
				</div>
				<Dropdown
					v-if="canWrite"
					v-cy="'taskDetail.addProperty'"
					eager
					class="add-property d-print-none"
				>
					<template #trigger="{toggleOpen, open}">
						<BaseButton
							class="dropdown-trigger add-property__trigger"
							:aria-expanded="open"
							@click="toggleOpen"
						>
							<Icon icon="plus" />
							{{ $t('task.detail.addProperty') }}
						</BaseButton>
					</template>
					<template #default="{close}">
						<template
							v-for="(group, index) in PROPERTY_ITEMS"
							:key="index"
						>
							<hr
								v-if="index > 0"
								class="dropdown-divider"
							>
							<span
								v-if="group.title"
								class="dropdown-heading"
							>{{ $t(group.title) }}</span>
							<DropdownItem
								v-for="item in group.items"
								:key="item.field"
								v-shortcut="item.shortcut ?? ''"
								v-cy="item.field === 'assignees' ? 'taskDetail.assign' : ''"
								:icon="item.icon"
								@click="close(); setFieldActive(item.field)"
							>
								{{ $t(`task.detail.actions.${item.label}`) }}
								<Shortcut
									is="span"
									v-if="item.shortcut"
									aria-hidden="true"
									:keys="shortcutBindingToDisplay(item.shortcut).keys"
								/>
							</DropdownItem>
						</template>
					</template>
				</Dropdown>
			</div>

			<ChecklistSummary :task="task" />

			<!-- Description -->
			<div class="details content description">
				<Description
					:model-value="task"
					:can-write="canWrite"
				/>
			</div>

			<!-- Reactions -->
			<Reactions
				:model-value="task.reactions"
				entity-kind="tasks"
				:entity-id="task.id"
				class="details d-print-none"
				:disabled="!canWrite"
			/>

			<!-- Attachments -->
			<div
				v-show="activeFields.attachments || hasAttachments"
				class="content attachments"
			>
				<Attachments
					:ref="e => { setFieldRef('attachments', e); attachmentsRef = e as any }"
					:edit-enabled="canWrite"
					:task="task"
				/>
			</div>

			<!-- Time Tracking -->
			<div
				v-if="timeTrackingEnabled && activeFields.timeTracking"
				:ref="e => setFieldRef('timeTracking', e)"
				class="content time-tracking"
			>
				<TaskTimeTracking :task-id="task.id" />
			</div>

			<!-- Related Tasks -->
			<div
				v-if="activeFields.relatedTasks"
				class="content details mbe-0"
			>
				<h2 class="task-section-title">
					<span class="icon is-grey">
						<Icon icon="sitemap" />
					</span>
					{{ $t('task.attributes.relatedTasks') }}
				</h2>
				<RelatedTasks
					:ref="e => setFieldRef('relatedTasks', e)"
					:edit-enabled="canWrite"
					:initial-related-tasks="task.related_tasks"
					:project-id="task.project_id"
					:show-no-relations-notice="true"
					:task-id="taskId"
				/>
			</div>

			<!-- Move Task -->
			<div
				v-if="activeFields.moveProject"
				class="content details"
			>
				<h2 class="task-section-title">
					<span class="icon is-grey">
						<Icon icon="list" />
					</span>
					{{ $t('task.detail.move') }}
				</h2>
				<div class="field has-addons">
					<div class="control is-expanded">
						<ProjectSearch
							:ref="e => setFieldRef('moveProject', e)"
							show-empty
							:filter="project => project.id !== task.project_id"
							@update:modelValue="changeProject"
						/>
					</div>
				</div>
			</div>

			<!-- Comments -->
			<Comments
				:can-write="canWrite"
				:task-id="taskId"
				:project-id="task.project_id"
			/>

			<!-- Marker element for scroll-to-bottom button visibility -->
			<div
				ref="contentBottomMarker"
				class="content-bottom-marker"
			/>
		</div>

		<BaseButton
			v-if="showScrollToCommentsButton"
			v-tooltip="$t('task.detail.scrollToBottom')"
			class="scroll-to-comments-button d-print-none"
			:aria-label="$t('task.detail.scrollToBottom')"
			@click="scrollToBottom"
		>
			<Icon icon="chevron-down" />
		</BaseButton>

		<Modal
			:enabled="showDeleteModal"
			@close="showDeleteModal = false"
			@submit="deleteTask()"
		>
			<template #header>
				<span>{{ $t('task.detail.delete.header') }}</span>
			</template>

			<template #text>
				<p class="tw:text-balance">
					{{ $t('task.detail.delete.text1') }}
				</p>
				<p class="tw:text-balance">
					{{ $t('task.detail.delete.text2') }}
				</p>
			</template>
		</Modal>
	</div>
</template>

<script lang="ts" setup>
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {ref, reactive, computed, watch, nextTick, onBeforeUnmount, onMounted, useTemplateRef, type ComponentPublicInstance} from 'vue'
import {useRouter, useRoute, type RouteLocation, onBeforeRouteLeave} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {unrefElement, useDebounceFn, useElementSize, useIntersectionObserver, useMutationObserver} from '@vueuse/core'
import {klona} from 'klona/lite'

import {useTask} from '@/composables/useTask'
import {canAutoFocus} from '@/directives/focus'
import {createTaskDraft, mergeTask} from '@/helpers/task'

import type {Task as ITask} from '@/client/generated'
import type {IconProp} from '@fortawesome/fontawesome-svg-core'
import type {ProjectResponse} from '@/client/queries/projects'

import {PRIORITIES} from '@/constants/priorities'
import {PERMISSIONS} from '@/constants/permissions'
import {PRO_FEATURE} from '@/constants/proFeatures'
import {SHORTCUTS} from '@/constants/shortcuts'
import {shortcutBindingToDisplay} from '@/helpers/shortcut'

import BaseButton from '@/components/base/BaseButton.vue'
import Dropdown from '@/components/misc/Dropdown.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import Shortcut from '@/components/misc/Shortcut.vue'

// partials
import Attachments from '@/components/tasks/partials/Attachments.vue'
import TaskTimeTracking from '@/components/time-tracking/TaskTimeTracking.vue'
import ChecklistSummary from '@/components/tasks/partials/ChecklistSummary.vue'
import ColorPicker from '@/components/input/ColorPicker.vue'
import Comments from '@/components/tasks/partials/Comments.vue'
import CreatedUpdated from '@/components/tasks/partials/CreatedUpdated.vue'
import Datepicker from '@/components/input/Datepicker.vue'
import Description from '@/components/tasks/partials/Description.vue'
import EditAssignees from '@/components/tasks/partials/EditAssignees.vue'
import EditLabels from '@/components/tasks/partials/EditLabels.vue'
import Heading from '@/components/tasks/partials/Heading.vue'
import ProjectSearch from '@/components/tasks/partials/ProjectSearch.vue'
import PercentDoneSelect from '@/components/tasks/partials/PercentDoneSelect.vue'
import PrioritySelect from '@/components/tasks/partials/PrioritySelect.vue'
import SizeSelect from '@/components/tasks/partials/SizeSelect.vue'
import type {TaskSize} from '@/constants/taskSizes'
import RelatedTasks from '@/components/tasks/partials/RelatedTasks.vue'
import Reminders from '@/components/tasks/partials/Reminders.vue'
import RepeatAfter from '@/components/tasks/partials/RepeatAfter.vue'
import TaskSubscription from '@/components/misc/Subscription.vue'
import CustomTransition from '@/components/misc/CustomTransition.vue'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'
import BucketSelect from '@/components/tasks/partials/BucketSelect.vue'
import Reactions from '@/components/input/Reactions.vue'

import {getProjectTitle} from '@/helpers/getProjectTitle'
import {scrollIntoView} from '@/helpers/scrollIntoView'
import {getScrollParent} from '@/helpers/getScrollParent'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {REMINDER_PERIOD_RELATIVE_TO_TYPES} from '@/types/IReminderPeriodRelativeTo'
import {playPopSound} from '@/helpers/playPop'
import {taskLoadErrorAction} from './taskDetailError'

import {
	useUpdateTaskMutation,
	useDeleteTaskMutation,
	useFavoriteTaskMutation,
	useDuplicateTaskMutation,
	useMarkTaskReadMutation,
} from '@/client/queries/taskMutations'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {useBaseStore} from '@/stores/base'
import {useConfigStore} from '@/stores/config'

import {useTitle} from '@/composables/useTitle'
import {useTaskDetailShortcuts} from '@/composables/useTaskDetailShortcuts'
import {useDelayedLoading} from '@/composables/useDelayedLoading'

import {error, success} from '@/message'
import type {Action as MessageAction} from '@/message'
import {useSetTaskSubscriptionMutation} from '@/client/queries/subscriptions'

const props = defineProps<{
	taskId: number,
	backdropView?: RouteLocation['fullPath'],
}>()

const router = useRouter()
const route = useRoute()
const {t} = useI18n({useScope: 'global'})

const projectList = useProjects()
const updateTask = useUpdateTaskMutation()
const deleteTaskMutation = useDeleteTaskMutation()
const favoriteTask = useFavoriteTaskMutation()
const duplicateTask = useDuplicateTaskMutation()
const markTaskRead = useMarkTaskReadMutation()
const subscriptionMutation = useSetTaskSubscriptionMutation()
const taskMutating = computed(() => [
	updateTask,
	deleteTaskMutation,
	favoriteTask,
	duplicateTask,
	markTaskRead,
].some(mutation => mutation.isPending.value))
const configStore = useConfigStore()
const authStore = useAuthStore()
const timeTrackingEnabled = computed(() => configStore.isProFeatureEnabled(PRO_FEATURE.TIME_TRACKING)
	&& !authStore.isLinkShareAuth)
const baseStore = useBaseStore()

const taskQuery = useTask(
	() => props.taskId ?? 0,
	() => [
		'reactions',
		'is_unread',
		'buckets',
		...(timeTrackingEnabled.value ? ['time_entries_count' as const] : []),
	],
)
const task = ref(createTaskDraft())

// Only fields edited here stay local; the rest follows the cache.
function followServerFields(loaded: ITask) {
	const {priority, size, percent_done, due_date, start_date, end_date, reminders, repeat_after, repeat_mode} = task.value
	task.value = {
		...createTaskDraft(klona(loaded)),
		priority,
		size,
		percent_done,
		due_date,
		start_date,
		end_date,
		reminders,
		repeat_after,
		repeat_mode,
	}
}

function seedTask(loaded: ITask) {
	task.value = createTaskDraft(klona(loaded))
	taskColor.value = task.value.hex_color ?? ''
}

const dueDateInput = computed({
	get: () => parseDateOrNull(task.value.due_date),
	set: (date: Date | string | null) => {
		task.value.due_date = parseDateOrNull(date)?.toISOString() ?? ''
	},
})
const startDateInput = computed({
	get: () => parseDateOrNull(task.value.start_date),
	set: (date: Date | string | null) => {
		task.value.start_date = parseDateOrNull(date)?.toISOString() ?? ''
	},
})
const endDateInput = computed({
	get: () => parseDateOrNull(task.value.end_date),
	set: (date: Date | string | null) => {
		task.value.end_date = parseDateOrNull(date)?.toISOString() ?? ''
	},
})
const hasAttachments = computed(() => (task.value.attachments?.length ?? 0) > 0)
const remindersDefaultRelativeTo = computed(() => {
	if (parseDateOrNull(task.value.due_date)) {
		return REMINDER_PERIOD_RELATIVE_TO_TYPES.DUEDATE
	}
	if (parseDateOrNull(task.value.start_date)) {
		return REMINDER_PERIOD_RELATIVE_TO_TYPES.STARTDATE
	}
	if (parseDateOrNull(task.value.end_date)) {
		return REMINDER_PERIOD_RELATIVE_TO_TYPES.ENDDATE
	}
	return null
})
const taskNotFound = ref(false)
const taskTitle = computed(() => task.value.title)
useTitle(taskTitle)

const lastProject = computed(() => {
	const backRoute = router.options.history.state?.back
	if (!backRoute || typeof backRoute !== 'string') {
		return null
	}

	const projectMatch = backRoute.match(/\/projects\/(-?\d+)/)
	if (!projectMatch || !projectMatch[1]) {
		return null
	}

	const id = parseInt(projectMatch[1])

	return projectList.projects[id] ?? null
})

const lastProjectOrTaskProject = computed(() => lastProject.value ?? project.value)

onBeforeRouteLeave(async () => {
	if (taskNotFound.value) {
		return
	}

	if (!lastProjectOrTaskProject.value) {
		await new Promise<void>((resolve) => {
			const timeout = setTimeout(() => {
				stop()
				resolve()
			}, 5000) // 5 second timeout

			const stop = watch(lastProjectOrTaskProject, (p) => {
				if (p) {
					clearTimeout(timeout)
					stop()
					resolve()
				}
			})
		})
	}

	if (lastProjectOrTaskProject.value) {
		baseStore.setCurrentProjectIfNotSet(lastProjectOrTaskProject.value)
	}
})

// We doubled the task color property here because verte does not have a real change property, leading
// to the color property change being triggered when the # is removed from it, leading to an update,
// which leads in turn to a change... This creates an infinite loop in which the task is updated, changed,
// updated, changed, updated and so on.
// To prevent this, we put the task color property in a separate value which is set to the task color
// when it is saved and loaded.
const taskColor = ref('')

// Used to avoid flashing of empty elements if the task content is not yet loaded.
const visible = ref(false)

const project = computed(() => projectList.projects[task.value.project_id ?? 0])

const projectRoute = computed(() => ({
	name: 'project.index',
	params: {projectId: task.value.project_id},
	hash: route.hash,
}))

const canWrite = computed(() => (
	(taskQuery.task.value?.max_permission ?? 0) > PERMISSIONS.READ
))

const isModal = computed(() => Boolean(props.backdropView))

const heading = ref<HTMLElement | null>(null)

async function scrollToHeading() {
	scrollIntoView(unrefElement(heading))
}

const attachmentsRef = ref<InstanceType<typeof Attachments> | null>(null)

const taskViewContainer = ref<HTMLElement | null>(null)
const scrollContainer = ref<HTMLElement | null>(null)
const contentBottomMarker = ref<HTMLElement | null>(null)
const bottomMarkerVisible = ref(true)
const isScrollable = ref(false)

function resolveScrollContainer() {
	// Null after unmount; deferred callers must not fall back to `document` then.
	if (!taskViewContainer.value) {
		return
	}

	scrollContainer.value = getScrollParent(taskViewContainer.value)
}

function updateScrollable() {
	const scroller = scrollContainer.value
	if (!scroller) {
		isScrollable.value = false
		return
	}

	isScrollable.value = scroller.scrollHeight > scroller.clientHeight + 1
}

const showScrollToCommentsButton = computed(() => {
	return isScrollable.value && !bottomMarkerVisible.value
})

function scrollToBottom() {
	if (!contentBottomMarker.value) {
		return
	}

	contentBottomMarker.value.scrollIntoView({
		behavior: 'smooth',
		block: 'end',
		inline: 'nearest',
	})
}

useIntersectionObserver(
	contentBottomMarker,
	([entry]) => {
		bottomMarkerVisible.value = entry?.isIntersecting ?? true
	},
	{threshold: 0.1},
)

const debouncedMutationHandler = useDebounceFn(async () => {
	await nextTick()
	resolveScrollContainer()
	updateScrollable()
}, 100)

useMutationObserver(
	taskViewContainer,
	debouncedMutationHandler,
	{subtree: true, childList: true},
)
onBeforeUnmount(() => debouncedMutationHandler.cancel())

const {height: scrollContainerHeight} = useElementSize(scrollContainer)
watch(scrollContainerHeight, () => updateScrollable())

onMounted(async () => {
	await nextTick()
	resolveScrollContainer()
	updateScrollable()
})

const taskBusy = useDelayedLoading(() => taskQuery.isFetching.value || taskMutating.value)


type FieldType =
	| 'assignees'
	| 'attachments'
	| 'color'
	| 'dueDate'
	| 'endDate'
	| 'labels'
	| 'moveProject'
	| 'percentDone'
	| 'priority'
	| 'relatedTasks'
	| 'reminders'
	| 'repeatAfter'
	| 'size'
	| 'startDate'
	| 'timeTracking'

// '+ Property' menu: field, label key (task.detail.actions.*), icon, shortcut
const PROPERTY_ITEMS: {title?: string, items: {field: FieldType, label: string, icon: IconProp, shortcut?: string}[]}[] = [
	{title: 'task.detail.organization', items: [
		{field: 'assignees', label: 'assign', icon: 'users', shortcut: SHORTCUTS.taskDetail.assignees},
		{field: 'labels', label: 'label', icon: 'tags', shortcut: SHORTCUTS.taskDetail.labels},
		{field: 'priority', label: 'priority', icon: 'exclamation-circle', shortcut: SHORTCUTS.taskDetail.priority},
	]},
	{title: 'task.detail.dateAndTime', items: [
		{field: 'dueDate', label: 'dueDate', icon: 'calendar', shortcut: SHORTCUTS.taskDetail.dueDate},
		{field: 'startDate', label: 'startDate', icon: 'play'},
		{field: 'endDate', label: 'endDate', icon: 'stop'},
		{field: 'reminders', label: 'reminders', icon: ['far', 'clock'], shortcut: SHORTCUTS.taskDetail.reminder},
		{field: 'repeatAfter', label: 'repeatAfter', icon: 'history'},
	]},
	{items: [
		{field: 'size', label: 'size', icon: 'ruler-horizontal'},
		{field: 'percentDone', label: 'percentDone', icon: 'percent'},
		{field: 'color', label: 'color', icon: 'fill-drip', shortcut: SHORTCUTS.taskDetail.color},
	]},
]

const activeFields: { [type in FieldType]: boolean } = reactive({
	assignees: false,
	attachments: false,
	color: false,
	dueDate: false,
	endDate: false,
	labels: false,
	moveProject: false,
	percentDone: false,
	priority: false,
	relatedTasks: false,
	reminders: false,
	repeatAfter: false,
	size: false,
	startDate: false,
	timeTracking: false,
})

function setActiveFields() {
	// Set all active fields based on values in the model
	activeFields.assignees = (task.value.assignees?.length ?? 0) > 0
	activeFields.attachments = (task.value.attachments?.length ?? 0) > 0
	activeFields.timeTracking = (task.value.time_entries_count ?? 0) > 0
	activeFields.dueDate = Boolean(parseDateOrNull(task.value.due_date))
	activeFields.endDate = Boolean(parseDateOrNull(task.value.end_date))
	activeFields.labels = (task.value.labels?.length ?? 0) > 0
	activeFields.percentDone = (task.value.percent_done ?? 0) > 0
	activeFields.priority = task.value.priority !== PRIORITIES.UNSET
	activeFields.relatedTasks = Object.keys(task.value.related_tasks ?? {}).length > 0
	activeFields.reminders = (task.value.reminders?.length ?? 0) > 0
	activeFields.repeatAfter = (task.value.repeat_after ?? 0) > 0
		|| task.value.repeat_mode !== TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT
	activeFields.size = task.value.size !== 0
	activeFields.startDate = Boolean(parseDateOrNull(task.value.start_date))
}

watch(taskQuery.task, async (loaded, previous) => {
	if (!loaded) return
	if (loaded.id !== previous?.id) {
		seedTask(loaded)
		setActiveFields()
	} else {
		followServerFields(loaded)
	}
	if (loaded.is_unread) markTaskRead.mutateAsync(loaded.id).catch(() => {})
	if (lastProject.value) baseStore.setCurrentProjectIfNotSet(lastProject.value)
	await nextTick()
	if (loaded.id !== previous?.id) scrollToHeading()
	resolveScrollContainer()
	updateScrollable()
	visible.value = true
}, {immediate: true})
watch(taskQuery.error, cause => {
	if (!cause) return
	const action = taskLoadErrorAction(cause)
	if (action === 'ignore') return
	if (action === 'notFound') {
		taskNotFound.value = true
		router.replace({name: 'not-found'})
		return
	}
	error(cause)
	visible.value = true
})

const activeFieldElements: { [id in FieldType]: HTMLElement | null } = reactive({
	assignees: null,
	attachments: null,
	color: null,
	dueDate: null,
	endDate: null,
	labels: null,
	moveProject: null,
	percentDone: null,
	priority: null,
	relatedTasks: null,
	reminders: null,
	repeatAfter: null,
	size: null,
	startDate: null,
	timeTracking: null,
})

const dueDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('dueDatePicker')
const startDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('startDatePicker')
const endDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('endDatePicker')

function setFieldRef(name: FieldType, e: Element | ComponentPublicInstance | null) {
	const element = e instanceof Element ? e : e?.$el
	activeFieldElements[name] = element instanceof HTMLElement ? element : null
}

function setFieldActive(fieldName: keyof typeof activeFields) {
	activeFields[fieldName] = true
	nextTick(() => {
		let datepicker: InstanceType<typeof Datepicker> | null = null
		switch (fieldName) {
			case 'dueDate':
				datepicker = dueDatePicker.value
				break
			case 'startDate':
				datepicker = startDatePicker.value
				break
			case 'endDate':
				datepicker = endDatePicker.value
				break
		}
		const el: HTMLElement | null = datepicker?.$el ?? activeFieldElements[fieldName]

		if (!el) {
			return
		}

		// On mobile the picker would pop up the keyboard.
		if (canAutoFocus() || !['labels', 'assignees', 'moveProject'].includes(fieldName)) {
			el.focus()
		}

		// Finish scrolling before the datepicker sheet locks the page.
		scrollIntoView(el, datepicker ? 'instant' : 'smooth')

		datepicker?.open()
	})
}

function openAttachments() {
	activeFields.attachments = true
	nextTick(() => {
		const el = activeFieldElements.attachments
		if (el) {
			scrollIntoView(el)
		}
		attachmentsRef.value?.openFilePicker()
	})
}

async function saveTask(
	currentTask: ITask | null = null,
	undoCallback?: () => void,
) {
	if (currentTask === null) {
		currentTask = klona(task.value)
	}

	if (!canWrite.value) {
		return
	}

	currentTask.hex_color = taskColor.value

	// If no end date is being set, but a start date and due date,
	// use the due date as the end date
	if (
		!parseDateOrNull(currentTask.end_date) &&
		Boolean(parseDateOrNull(currentTask.start_date)) &&
		Boolean(parseDateOrNull(currentTask.due_date))
	) {
		currentTask.end_date = currentTask.due_date
	}

	seedTask(mergeTask(task.value, await updateTask.mutateAsync({...currentTask, id: currentTask.id!})))
	setActiveFields()

	let actions: MessageAction[] = []
	if (undoCallback) {
		actions = [{
			title: t('task.undo'),
			callback: undoCallback,
		}]
	}
	success({message: t('task.detail.updateSuccess')}, actions)
}

useTaskDetailShortcuts({
	task: () => task.value,
	taskTitle: () => taskTitle.value,
	onSave: saveTask,
})

const showDeleteModal = ref(false)

async function deleteTask() {
	await deleteTaskMutation.mutateAsync(task.value.id!)
	success({message: t('task.detail.deleteSuccess')})
	router.push({name: 'project.index', params: {projectId: task.value.project_id}})
}

async function toggleTaskDone() {
	const newTask = {
		...task.value,
		done: !task.value.done,
	}

	if (newTask.done) {
		playPopSound()
	}

	await saveTask(
		newTask,
		toggleTaskDone,
	)
}

async function changeProject(project: ProjectResponse | null) {
	if (project === null) {
		return
	}
	await saveTask({
		...task.value,
		project_id: project.id,
	})
	baseStore.setCurrentProject(project)
}

function toggleSubscription(subscribed: boolean) {
	if (subscriptionMutation.isPending.value) return
	subscriptionMutation.mutate({taskId: task.value.id!, subscribed})
}

async function toggleFavorite() {
	await favoriteTask.mutateAsync({...task.value, id: task.value.id!})
}

async function duplicateCurrentTask() {
	const duplicatedTask = await duplicateTask.mutateAsync(task.value.id!)
	if (duplicatedTask) {
		success({message: t('task.detail.duplicateSuccess')})
		router.push({
			name: 'task.detail',
			params: {id: duplicatedTask.id},
		})
	}
}

async function setPriority(priority: number) {
	const newTask: ITask = {
		...task.value,
		priority,
	}

	return saveTask(newTask)
}

async function setSize(size: TaskSize) {
	const newTask: ITask = {
		...task.value,
		size,
	}

	return saveTask(newTask)
}

async function setPercentDone(percentDone: number) {
	const newTask: ITask = {
		...task.value,
		percent_done: percentDone,
	}

	return saveTask(newTask)
}

async function removeRepeatAfter() {
	task.value.repeat_after = 0
	task.value.repeat_mode = TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT
	await saveTask()
}

function setRelatedTasksActive() {
	setFieldActive('relatedTasks')

	// If the related tasks are already available, show the form again
	const el = activeFieldElements['relatedTasks']
	if (!el) {
		return
	}
	for (const child of Array.from(el.children)) {
		if ((child as HTMLElement).id === 'showRelatedTasksFormButton') {
			(child as HTMLElement).click()
			break
		}
	}
}
</script>

<style lang="scss" scoped>
.task-view-container {
	// simulate sass lighten($primary, 30) by increasing lightness 30% to 73%
	--primary-light: hsla(var(--primary-h), var(--primary-s), 73%, var(--primary-a));
	padding-block-end: 0;

	@media screen and (min-width: $desktop) {
		padding-block-end: 1rem;
	}
}

.task-view {
	padding-block-start: 1rem;
	padding-inline: .5rem;
	background-color: var(--site-background);

	@media screen and (min-width: $desktop) {
		padding: 1rem;
	}
}

.is-modal .task-view {
	border-radius: $radius;
	padding: 1rem;
	color: var(--text);
	background-color: var(--site-background) !important;

	@media screen and (max-width: $tablet) {
		border-radius: 0;
	}
}

.task-view * {
	transition: opacity 50ms ease;
}

.is-loading .task-view * {
	opacity: 0;
}


.task-topbar {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
	margin-block-end: .5rem;
}

// keep the actions clear of the modal's close button in its corner
.is-modal .task-topbar {
	padding-inline-end: 2.5rem;
}

.task-topbar__actions {
	display: flex;
	align-items: center;
	gap: .25rem;
	margin-inline-start: auto;
	color: var(--grey-500);

	:deep(.dropdown-trigger) {
		padding: .375rem;
	}
}

.topbar-button {
	padding: .375rem;
	border-radius: $radius;
	transition: color $transition;

	&:hover,
	&.is-favorite {
		color: var(--warning);
	}
}

.task-title-row {
	display: flex;
	align-items: flex-start;
	gap: .5rem;
}

.done-toggle {
	flex: none;
	display: flex;
	align-items: center;
	justify-content: center;
	inline-size: 1.75rem;
	block-size: 1.75rem;
	margin-block-start: .45rem; // centre on the first line of the title
	border: 2px solid var(--grey-300);
	border-radius: 100%;
	color: transparent;
	transition: all $transition;

	&:hover,
	&:focus-visible {
		border-color: var(--success);
		color: var(--success);
	}

	// bright brand green with fixed dark text passes contrast in both themes
	&.done-toggle--done {
		background-color: var(--success);
		border-color: var(--success);
		color: hsl(215, 27.9%, 16.9%);
	}
}

.properties {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .375rem;
	margin-block: .75rem;
	padding-block-end: .75rem;
	border-block-end: 1px solid var(--grey-200);
}

.property {
	display: inline-flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .375rem;
	max-inline-size: 100%;
	min-inline-size: 0;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	padding: .125rem .5rem;

	.detail-title {
		font-size: .75rem;
		color: var(--grey-500);
		white-space: nowrap;
	}

	&.assignees,
	&.labels-list {
		flex: 1 1 16rem;
	}
}

.add-property__trigger {
	display: inline-flex;
	align-items: center;
	gap: .25rem;
	border: 1px dashed var(--grey-300);
	border-radius: $radius;
	padding: .25rem .625rem;
	font-size: .875rem;
	color: var(--grey-500);

	&:hover,
	&[aria-expanded="true"] {
		border-color: var(--link);
		color: var(--link);
	}
}

.dropdown-heading {
	display: block;
	padding: .25rem 1rem;
	text-transform: uppercase;
	color: var(--grey-500);
	font-size: .75rem;
	font-weight: 700;
}

// push the shortcut hint to the end of each '+ Property' item
.add-property :deep(.dropdown-item > span:last-child) {
	display: flex;
	flex: 1;
	justify-content: space-between;
	gap: 1rem;
}

.subtitle {
	color: var(--grey-500);
	margin-block-end: 0;

	a {
		color: var(--grey-800);
	}
}

h2 .button {
	vertical-align: middle;
}

.icon.is-grey {
	color: var(--grey-400);
}

.date-input {
	display: flex;
	align-items: center;
}

.remove {
	color: var(--danger);
	vertical-align: middle;
	padding-inline-start: .5rem;
	line-height: 1;
}

:deep(.datepicker) {
	inline-size: 100%;

	.show {
		color: var(--text);
		padding: .25rem .5rem;
		transition: background-color $transition;
		border-radius: $radius;
		display: block;
		margin: .1rem 0;
		inline-size: 100%;
		text-align: start;

		&:hover {
			background: var(--white);
		}
	}

	&.disabled .show:hover {
		background: transparent;
	}
}

.details {
	padding-block-end: 0.75rem;
	margin-block-end: 0;

	.none {
		font-style: italic;
	}
}

.labels-list,
.assignees {
	:deep(.multiselect) {
		.input-wrapper {
			&:not(:focus-within, :hover) {
				background: transparent;
				border-color: transparent;
			}
		}
	}
}

:deep(.details),
:deep(.properties),
:deep(.heading) {
	.input:not(.has-defaults),
	.textarea,
	.select:not(.has-defaults) select {
		cursor: pointer;
		transition: all $transition-duration;

		&::placeholder {
			color: var(--text-light);
			opacity: 1;
			font-style: italic;
		}

		&:not(:disabled) {
			&:hover,
			&:active,
			&:focus {
				background: var(--scheme-main);
				border-color: var(--border);
				cursor: text;
			}

			&:hover,
			&:active {
				cursor: text;
				border-color: var(--link)
			}
		}
	}

	.select:not(.has-defaults):after {
		opacity: 0;
	}

	.select:not(.has-defaults):hover:after {
		opacity: 1;
	}
}

.attachments {
	margin-block-end: 0;

	table tr:last-child td {
		border-inline-end: none;
	}
}

.checklist-summary {
	padding-inline-start: .25rem;
}

// Mobile: keep the comment composer reachable while the comments are in view.
.task-view :deep(.comments > .media.comment.d-print-none:not(.new-comment-top)) {
	@media screen and (max-width: $tablet) {
		position: sticky;
		inset-block-end: 0;
		z-index: 2;
		background: var(--site-background);
	}
}

.scroll-to-comments-button {
	position: fixed;
	// Position above the keyboard shortcuts button (which is at bottom: calc(1rem - 4px))
	inset-block-end: 2.5rem;
	inset-inline-end: .75rem;
	z-index: 10;
	inline-size: 2rem;
	block-size: 2rem;
	border-radius: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 0;
	background-color: var(--site-background);
	border: 1px solid var(--grey-300);
	color: var(--grey-500);
	box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
	transition: all $transition;

	&:hover {
		background-color: var(--grey-100);
		color: var(--grey-700);
	}

	@media screen and (max-width: $tablet) {
		// Hide on mobile since keyboard shortcuts button is also hidden
		display: none;
	}
}
</style>

<style lang="scss">
// global style to override position when the modal task detail is active
.modal-content .scroll-to-comments-button {
	inset-block-end: .75rem;
	inset-inline-end: 1rem;
}
</style>
