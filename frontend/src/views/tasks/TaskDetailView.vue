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
				<!-- The fields the table edits inline are always here; the rest join once set, via + More. -->
				<TaskPropertyChip
					:ref="e => setChipRef('priority', e)"
					:removable="Boolean(task.priority)"
					:shortcut="SHORTCUTS.taskDetail.priority"
					close-on-change
					class="property"
					:label="$t('task.attributes.priority')"
					:can-write="canWrite"
					@remove="clearField('priority')"
				>
					<template #value>
						<span
							class="chip-priority"
							:data-priority="task.priority"
						>{{ $t(`task.priority.${PRIORITY_KEYS[task.priority ?? 0] ?? 'unset'}`) }}</span>
					</template>
					<PrioritySelect
						v-model="task.priority"
						@update:modelValue="setPriority"
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					:ref="e => setChipRef('size', e)"
					:removable="Boolean(task.size)"
					close-on-change
					class="property"
					:label="$t('task.attributes.size')"
					:can-write="canWrite"
					@remove="clearField('size')"
				>
					<template #value>
						{{ $t(taskSizeLabelKey(task.size ?? 0)) }}
					</template>
					<SizeSelect
						v-model="task.size"
						@update:modelValue="setSize"
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					:ref="e => setChipRef('assignees', e)"
					:removable="Boolean(task.assignees?.length)"
					:shortcut="SHORTCUTS.taskDetail.assignees"
					class="property assignees"
					:label="$t('task.attributes.assignees')"
					:can-write="canWrite"
					@remove="clearField('assignees')"
				>
					<template #value>
						<span
							v-if="task.assignees?.length"
							class="chip-avatars"
						>
							<User
								v-for="user in task.assignees.slice(0, 3)"
								:key="user.id"
								:user="user"
								:avatar-size="20"
								:show-username="false"
							/>
						</span>
						{{ assigneeSummary || '—' }}
					</template>
					<EditAssignees
						v-model="task.assignees"
						class="chip-editor"
						:project-id="task.project_id"
						:task-id="task.id"
						inline
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					:ref="e => setChipRef('labels', e)"
					:removable="Boolean(task.labels?.length)"
					:shortcut="SHORTCUTS.taskDetail.labels"
					class="property labels-list"
					:label="$t('task.attributes.labels')"
					:can-write="canWrite"
					@remove="clearField('labels')"
				>
					<template #value>
						<template v-if="!task.labels?.length">
							—
						</template>
						<span
							v-for="label in task.labels"
							:key="label.id"
							class="chip-label"
							:style="getLabelStyles(label)"
						>{{ label.title }}</span>
					</template>
					<EditLabels
						v-model="task.labels"
						class="chip-editor"
						:project-id="task.project_id"
						:task-id="taskId"
						:creatable="!authStore.isLinkShareAuth"
						:creation-disabled-message="authStore.isLinkShareAuth ? $t('task.label.linkShareCannotCreate') : ''"
						inline
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					:removable="parseDateOrNull(task.due_date) !== null"
					:shortcut="SHORTCUTS.taskDetail.dueDate"
					class="property date-input"
					:class="{'is-overdue': isOverdue}"
					:label="$t('task.attributes.dueDate')"
					:can-write="canWrite"
					:remove-label="$t('task.detail.removeDueDate')"
					@remove="clearField('dueDate')"
					@activate="dueDatePicker?.open()"
				>
					<template #value>
						<Datepicker
							ref="dueDatePicker"
							v-model="dueDateInput"
							:choose-date-label="$t('task.detail.chooseDueDate')"
							empty-label="—"
							:title="$t('task.attributes.dueDate')"
							:disabled="taskBusy || !canWrite"
							@closeOnChange="saveTask()"
						/>
					</template>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.startDate"
					class="property date-input"
					:label="$t('task.attributes.startDate')"
					:can-write="canWrite"
					:remove-label="$t('task.detail.removeStartDate')"
					@remove="clearField('startDate')"
					@activate="startDatePicker?.open()"
				>
					<template #value>
						<Datepicker
							ref="startDatePicker"
							v-model="startDateInput"
							:choose-date-label="$t('task.detail.chooseStartDate')"
							empty-label="—"
							:title="$t('task.attributes.startDate')"
							:disabled="taskBusy || !canWrite"
							@closeOnChange="saveTask()"
						/>
					</template>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.endDate"
					class="property date-input"
					:label="$t('task.attributes.endDate')"
					:can-write="canWrite"
					:remove-label="$t('task.detail.removeEndDate')"
					@remove="clearField('endDate')"
					@activate="endDatePicker?.open()"
				>
					<template #value>
						<Datepicker
							ref="endDatePicker"
							v-model="endDateInput"
							:choose-date-label="$t('task.detail.chooseEndDate')"
							empty-label="—"
							:title="$t('task.attributes.endDate')"
							:disabled="taskBusy || !canWrite"
							@closeOnChange="saveTask()"
						/>
					</template>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.percentDone"
					:ref="e => setChipRef('percentDone', e)"
					close-on-change
					class="property"
					:label="$t('task.attributes.percentDone')"
					:can-write="canWrite"
					@remove="clearField('percentDone')"
				>
					<template #value>
						<span
							class="chip-progress"
							aria-hidden="true"
						><span :style="{inlineSize: `${(task.percent_done ?? 0) * 100}%`}" /></span>
						{{ Math.round((task.percent_done ?? 0) * 100) }}%
					</template>
					<PercentDoneSelect
						v-model="task.percent_done"
						@update:modelValue="setPercentDone"
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.reminders"
					:ref="e => setChipRef('reminders', e)"
					class="property"
					:label="$t('task.attributes.reminders')"
					:can-write="canWrite"
					@remove="clearField('reminders')"
				>
					<template #value>
						<!-- Read-only users get no popover, so show the times themselves. -->
						{{ canWrite
							? task.reminders?.length || ''
							: (task.reminders ?? []).map(r => formatDateShort(r.reminder)).join(', ') }}
					</template>
					<Reminders
						v-model="task.reminders"
						:default-relative-to="remindersDefaultRelativeTo"
						@update:modelValue="saveTask()"
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.repeatAfter"
					:ref="e => setChipRef('repeatAfter', e)"
					class="property"
					:label="$t('task.attributes.repeat')"
					:can-write="canWrite"
					:remove-label="$t('task.detail.removeRepeat')"
					@remove="clearField('repeatAfter')"
				>
					<template #value>
						{{ repeatSummary }}
					</template>
					<RepeatAfter
						:model-value="task"
						@update:modelValue="saveTask($event)"
					/>
				</TaskPropertyChip>
				<TaskPropertyChip
					v-if="activeFields.color"
					:ref="e => setChipRef('color', e)"
					close-on-change
					class="property"
					:label="$t('task.attributes.color')"
					:can-write="canWrite"
					@remove="clearField('color')"
				>
					<template #value>
						<ColorBubble
							v-if="taskColor"
							class="chip-swatch"
							:color="getHexColor(taskColor)"
						/>
					</template>
					<ColorPicker
						v-model="taskColor"
						@update:modelValue="saveTask()"
					/>
				</TaskPropertyChip>
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
							{{ $t('task.detail.moreProperties') }}
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
			<div class="details content description reactions-host">
				<Description
					:model-value="task"
					:can-write="canWrite"
				>
					<!-- Without reactions, adding one is a quiet icon in the description header instead of a lone row. -->
					<template
						v-if="canWrite && !hasReactions"
						#actions
					>
						<Reactions
							:model-value="task.reactions"
							entity-kind="tasks"
							:entity-id="task.id"
							class="reactions--quiet description-reactions d-print-none"
						/>
					</template>
				</Description>
				<Reactions
					v-if="hasReactions"
					:model-value="task.reactions"
					entity-kind="tasks"
					:entity-id="task.id"
					class="details d-print-none"
					:disabled="!canWrite"
				/>
				<!-- No description header to hold it: tuck the add button into the collapsed description line. -->
				<Reactions
					v-else-if="canWrite && isEditorContentEmpty(task.description ?? '')"
					:model-value="task.reactions"
					entity-kind="tasks"
					:entity-id="task.id"
					class="reactions--quiet description-reactions--corner d-print-none"
				/>
			</div>

			<TaskSubtasks
				:task="task"
				:can-write="canWrite"
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
import {createTaskDraft, getHexColor, mergeTask, parseRepeatAfter} from '@/helpers/task'
import {getDisplayName} from '@/helpers/user'
import {formatDateShort} from '@/helpers/time/formatDate'
import {isEditorContentEmpty} from '@/helpers/editorContentEmpty'
import {useLabelStyles} from '@/composables/useLabelStyles'

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
import {taskSizeLabelKey, type TaskSize} from '@/constants/taskSizes'
import RelatedTasks from '@/components/tasks/partials/RelatedTasks.vue'
import Reminders from '@/components/tasks/partials/Reminders.vue'
import RepeatAfter from '@/components/tasks/partials/RepeatAfter.vue'
import TaskSubscription from '@/components/misc/Subscription.vue'
import TaskPropertyChip from '@/components/tasks/partials/TaskPropertyChip.vue'
import TaskSubtasks from '@/components/tasks/partials/TaskSubtasks.vue'
import User from '@/components/misc/User.vue'
import ColorBubble from '@/components/misc/ColorBubble.vue'
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
	useAddTaskAssigneeMutation,
	useAddTaskLabelMutation,
	useRemoveTaskAssigneeMutation,
	useRemoveTaskLabelMutation,
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
const addAssigneeMutation = useAddTaskAssigneeMutation()
const removeAssigneeMutation = useRemoveTaskAssigneeMutation()
const addLabelMutation = useAddTaskLabelMutation()
const removeLabelMutation = useRemoveTaskLabelMutation()
const {getLabelStyles} = useLabelStyles()
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
const hasReactions = computed(() => Object.keys(task.value.reactions ?? {}).length > 0)

// Chip values for the properties strip
const PRIORITY_KEYS = ['unset', 'low', 'medium', 'high', 'urgent', 'doNow']
const assigneeSummary = computed(() => {
	const assignees = task.value.assignees ?? []
	return assignees.length > 2
		? t('task.detail.assigneeCount', {count: assignees.length})
		: assignees.map(user => getDisplayName(user)).join(', ')
})
const isOverdue = computed(() => {
	const due = parseDateOrNull(task.value.due_date)
	return !task.value.done && due !== null && due < new Date()
})
const repeatSummary = computed(() => {
	if (task.value.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH) {
		return t('task.repeat.monthly')
	}
	const {amount, type} = parseRepeatAfter(task.value.repeat_after ?? 0)
	if (amount === 1 && (type === 'days' || type === 'weeks')) {
		return t(type === 'days' ? 'task.repeat.everyDay' : 'task.repeat.everyWeek')
	}
	// The editor only offers hours, days and weeks; other units only come from the API.
	return amount > 0 && ['hours', 'days', 'weeks'].includes(type)
		? `${t('task.repeat.each')} ${amount} ${t(`task.repeat.${type}`).toLowerCase()}`
		: ''
})
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
	{title: 'task.detail.dateAndTime', items: [
		{field: 'startDate', label: 'startDate', icon: 'play'},
		{field: 'endDate', label: 'endDate', icon: 'stop'},
		{field: 'reminders', label: 'reminders', icon: ['far', 'clock'], shortcut: SHORTCUTS.taskDetail.reminder},
		{field: 'repeatAfter', label: 'repeatAfter', icon: 'history'},
	]},
	{items: [
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
	activeFields.relatedTasks = Object.keys(task.value.related_tasks ?? {}).some(kind => kind !== 'subtask')
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

// Fields whose element setFieldActive focuses and scrolls to; chips handle their own.
const activeFieldElements: Partial<Record<FieldType, HTMLElement | null>> = reactive({})

const dueDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('dueDatePicker')
const startDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('startDatePicker')
const endDatePicker = useTemplateRef<InstanceType<typeof Datepicker>>('endDatePicker')

function setFieldRef(name: FieldType, e: Element | ComponentPublicInstance | null) {
	const element = e instanceof Element ? e : e?.$el
	activeFieldElements[name] = element instanceof HTMLElement ? element : null
}

// Property chips whose editor lives in a popover
const chipRefs: Partial<Record<FieldType, InstanceType<typeof TaskPropertyChip>>> = {}

function setChipRef(name: FieldType, e: Element | ComponentPublicInstance | null) {
	if (e instanceof Element || e === null) {
		delete chipRefs[name]
		return
	}
	chipRefs[name] = e as InstanceType<typeof TaskPropertyChip>
}

function setFieldActive(fieldName: keyof typeof activeFields) {
	activeFields[fieldName] = true
	nextTick(() => {
		const chip = chipRefs[fieldName]
		if (chip) {
			scrollIntoView(chip.$el, 'instant')
			chip.open()
			return
		}

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
		const el: HTMLElement | null = datepicker?.$el ?? activeFieldElements[fieldName] ?? null

		if (!el) {
			return
		}

		// On mobile the picker would pop up the keyboard.
		if (canAutoFocus() || fieldName !== 'moveProject') {
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
	// Like the back button: return to the project view it was opened from, keeping that view's filter and page.
	// A fresh project route would drop them.
	if (lastProject.value) {
		router.back()
	} else {
		router.push(projectRoute.value)
	}
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

// A chip's ×: clear the value like the old per-field remove buttons did, then hide the chip.
async function clearField(field: FieldType) {
	if (field === 'assignees' || field === 'labels') {
		return clearList(field)
	}
	activeFields[field] = false
	switch (field) {
		case 'priority':
			if (task.value.priority === PRIORITIES.UNSET) return
			return setPriority(PRIORITIES.UNSET)
		case 'size':
			if (!task.value.size) return
			return setSize(0)
		case 'percentDone':
			if (!task.value.percent_done) return
			return setPercentDone(0)
		case 'reminders':
			if (!task.value.reminders?.length) return
			task.value.reminders = []
			return saveTask()
		case 'repeatAfter':
			if (!task.value.repeat_after && task.value.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_DEFAULT) return
			return removeRepeatAfter()
		case 'color':
			if (!taskColor.value) return
			taskColor.value = ''
			return saveTask()
		case 'dueDate':
		case 'startDate':
		case 'endDate': {
			const key = ({dueDate: 'due_date', startDate: 'start_date', endDate: 'end_date'} as const)[field]
			if (!task.value[key]) return
			task.value[key] = ''
			return saveTask()
		}
	}
}

// Removes every assignee or label one by one. The mutations update the task cache, which the
// draft follows, so there is nothing to write back here (and the open task may have changed by then).
async function clearList(field: 'assignees' | 'labels') {
	const taskId = task.value.id!
	const assignees = field === 'assignees' ? [...(task.value.assignees ?? [])] : []
	const labels = field === 'labels' ? [...(task.value.labels ?? [])] : []
	try {
		for (const user of assignees) {
			await removeAssigneeMutation.mutateAsync({user: {...user, id: user.id!}, taskId})
		}
		for (const label of labels) {
			await removeLabelMutation.mutateAsync({label: {...label, id: label.id!}, taskId})
		}
	} catch {
		// The mutation already showed the error; keep the chip so the rest can be retried.
		return
	}
	if (task.value.id === taskId) {
		activeFields[field] = false
	}
	success({message: t('task.detail.updateSuccess')}, [{
		title: t('task.undo'),
		callback: async () => {
			try {
				for (const user of assignees) {
					await addAssigneeMutation.mutateAsync({user: {...user, id: user.id!}, taskId})
				}
				for (const label of labels) {
					await addLabelMutation.mutateAsync({label: {...label, id: label.id!}, taskId})
				}
			} catch {
				// Toasted by the mutation.
			}
			if (task.value.id === taskId) {
				activeFields[field] = true
			}
		},
	}])
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
	background-color: var(--white) !important;

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
	align-items: center;
	gap: .5rem;
	margin-block-end: .5rem;
}

// keep the actions clear of the modal's close button in its corner
.is-modal .task-topbar {
	padding-inline-end: 2.5rem;
}

.task-topbar__actions {
	flex: none;
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
	margin-block-start: .6rem; // centre on the first line of the title

	@media screen and (max-width: $tablet) {
		margin-block-start: .35rem;
	}
	border: 2px solid var(--grey-300);
	border-radius: 100%;
	color: var(--grey-400);
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
	display: grid;
	grid-template-columns: repeat(4, minmax(0, 1fr));
	gap: .5rem;

	@media screen and (max-width: $tablet) {
		grid-template-columns: repeat(2, minmax(0, 1fr));
	}

	margin-block: .875rem;
	padding-block-end: 1.125rem;
	border-block-end: 1px solid var(--grey-200);
}

// Line the chips up with the title text, past the done circle.
.task-title-row:has(.done-toggle) + .properties {
	@media screen and (min-width: $tablet) {
		padding-inline-start: 2.6rem;
	}
}

.chip-avatars {
	display: inline-flex;

	> .user:not(:first-child) {
		margin-inline-start: -.375rem;
	}

	:deep(.avatar-wrapper) {
		margin-inline-end: 0;
	}

	:deep(.avatar) {
		box-sizing: content-box;
		border: 2px solid var(--grey-50);
	}
}

.chip-priority {
	&[data-priority="3"] {
		color: var(--priority-3-text);
	}

	&[data-priority="4"],
	&[data-priority="5"] {
		color: var(--priority-4-text);
	}
}

.chip-label {
	padding: .1875rem .5rem;
	border-radius: 1rem;
	font-size: .75rem;
}

.chip-progress {
	inline-size: 3rem;
	block-size: .375rem;
	border-radius: 3px;
	background: var(--grey-200);
	overflow: hidden;

	> span {
		display: block;
		block-size: 100%;
		background: var(--primary);
	}
}

.chip-swatch {
	inline-size: .875rem;
	block-size: .875rem;
}

// Popover editors: the search field on top, its results right under it, the chosen items below.
.chip-editor.multiselect {
	display: flex;
	flex-wrap: wrap;
	gap: .375rem;
	// The popover sizes to its content; the search editors get a fixed ~300px.
	inline-size: 18.5rem;

	.bottom-sheet & {
		inline-size: 100%;
	}

	:deep(.control),
	:deep(.input-wrapper) {
		display: contents;
	}

	:deep(.input-wrapper .input) {
		flex: 1 0 100%;
		max-inline-size: none !important;
		min-block-size: 2.25rem;
		padding-inline: calc(.75em - 1px);
		border: 1px solid var(--input-border-color) !important;
		border-radius: $radius;
		background: var(--white);
		font-size: .8125rem;

		&:focus {
			border-color: var(--primary) !important;
		}
	}

	&.has-search-results :deep(.input-wrapper .input) {
		border-color: var(--primary) !important;
		border-radius: $radius $radius 0 0;
	}

	:deep(.search-results) {
		flex: 1 0 100%;
		margin-block-start: calc(-.375rem - 1px);
	}

	// labels' tags and the assignee list
	:deep(.input-wrapper > :not(.input)) {
		order: 1;
	}

	:deep(.assignees-list) {
		flex: 1 0 100%;
		flex-direction: column;
	}

	:deep(.assignees-list .assignee) {
		display: flex;
		align-items: center;
		gap: .5rem;
		margin: 0 !important;
		padding: .25rem;
		border-radius: $radius;

		&:hover {
			background: var(--grey-100);
		}
	}

	:deep(.assignees-list .user) {
		flex: 1;
		min-inline-size: 0;
		margin: 0 !important;
	}

	:deep(.assignees-list .avatar) {
		inline-size: 1.5rem;
		block-size: 1.5rem;
		border: 0;
	}

	// AssigneeList hides the names on desktop
	:deep(.assignees-list .user .username) {
		display: inline;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: .875rem;
	}

	:deep(.remove-assignee) {
		position: static;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex: none;
		inline-size: 1.5rem;
		block-size: 1.5rem;
		border-radius: $radius;
		background: transparent;
		color: var(--text-muted);

		&:hover {
			background: var(--grey-200);
			color: var(--danger);
		}
	}
}

// The datepicker's own trigger is the chip's value.
.date-input :deep(.datepicker .show) {
	margin: 0;
	padding: 0;
	inline-size: auto;
	color: inherit;
	white-space: nowrap;

	&:hover {
		background: transparent;
	}
}

.property.is-overdue {
	border-color: color-mix(in srgb, var(--danger) 35%, transparent);
	background: color-mix(in srgb, var(--danger) 7%, var(--white));

	:deep(.datepicker .show) {
		color: var(--danger-text);
	}
}

.add-property {
	display: flex;
}

.add-property__trigger {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: .375rem;
	inline-size: 100%;
	min-block-size: 3rem;
	padding-inline: .75rem;
	border: 1px dashed var(--grey-300);
	border-radius: $radius;
	font-size: .8125rem;
	color: var(--grey-600);

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
	display: flex;
	align-items: center;
	gap: .25rem;
	min-inline-size: 0;
	white-space: nowrap;
	color: var(--grey-500);
	margin-block-end: 0;

	// Long project names truncate so ★ ⋯ stay on the breadcrumb's row.
	a {
		min-inline-size: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		color: var(--grey-800);
	}
}

// Borderless until hovered, like the mockup's collapsed description line.
.description :deep(.tiptap__collapsed) {
	border-color: transparent;

	&:hover {
		border-color: var(--input-border-color);
		background: var(--grey-50);
	}

	&:focus-visible {
		border-color: var(--primary);
	}
}

h2 .button {
	vertical-align: middle;
}

// Quiet add-reaction buttons (see .reactions--quiet in styles/components/tasks.scss)
.description-reactions {
	position: relative;
	display: inline-flex;

	:deep(.emoji-picker) {
		inset-block-start: 100%;
	}
}

.description {
	position: relative;
}

.description-reactions--corner {
	position: absolute;
	inset-block-start: .5rem;
	inset-inline-end: .5rem;

	// Only next to the collapsed line; an open editor has its toolbar there.
	.description:not(:has(.tiptap__collapsed)) & {
		display: none;
	}
}

.details {
	padding-block-end: 0.75rem;
	margin-block-end: 0;

	.none {
		font-style: italic;
	}
}

:deep(.details),
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
