<template>
	<div
		class="board-task loader-container"
		:class="{'is-loading': isLoading}"
		:data-task-id="task.id"
		:data-progress="progress"
		@click="openTaskDetail"
	>
		<span
			v-tooltip="!canMarkAsDone ? $t('task.readOnlyCheckbox') : ''"
			class="board-task__check"
		>
			<FancyCheckbox
				:model-value="task.done ?? false"
				:disabled="!canMarkAsDone"
				:aria-label="$t('task.detail.markAsDone', {task: task.title})"
				@update:modelValue="markAsDone"
				@click.stop
			/>
		</span>

		<div class="board-task__body">
			<!-- Dots and avatars trail the title text, so a task with nothing else stays one line tall. -->
			<div class="board-task__title-line">
				<TaskGlanceTooltip :task="task">
					<RouterLink
						:to="taskRoute"
						class="board-task__title"
					>
						<span
							v-if="priorityName"
							v-tooltip="priorityName"
							class="board-task__priority"
							role="img"
							:aria-label="priorityName"
						>!</span>
						{{ task.title }}
					</RouterLink>
				</TaskGlanceTooltip>
				<span
					v-for="label in labels"
					:key="label.id"
					v-tooltip="label.title"
					class="board-task__dot"
					:style="{backgroundColor: getHexColor(label.hex_color)}"
					role="img"
					:aria-label="label.title"
				/>
				<AssigneeList
					v-if="(task.assignees?.length ?? 0) > 0"
					class="board-task__assignees"
					:assignees="task.assignees ?? []"
					:avatar-size="20"
					inline
				/>
			</div>

			<div class="board-task__meta">
				<time
					v-if="dueDate"
					v-tooltip="formatDateLong(dueDate)"
					:datetime="formatISO(dueDate)"
					:class="{'is-overdue': isOverdue}"
				>{{ dueDateText }}</time>
				<CommentCount :task="task" />
				<TaskStatusSelect
					:task="task"
					:can-write="canEditStatus"
					hide-default-bucket
				/>
			</div>
		</div>
		<TaskQuickActions
			:task="task"
			:can-write="canEditStatus"
		/>
		<slot />
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import type {Task} from '@/client/generated'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'
import CommentCount from '@/components/tasks/partials/CommentCount.vue'
import TaskGlanceTooltip from '@/components/tasks/partials/TaskGlanceTooltip.vue'
import TaskQuickActions from '@/components/tasks/partials/TaskQuickActions.vue'
import TaskStatusSelect from '@/components/tasks/partials/TaskStatusSelect.vue'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {useGlobalNow} from '@/composables/useGlobalNow'
import {useMarkTaskDone} from '@/composables/useMarkTaskDone'
import {useTaskStatus} from '@/composables/useTaskStatus'
import {PRIORITIES} from '@/constants/priorities'
import {getHexColor} from '@/helpers/task'
import {formatDateLong, formatDisplayDate, formatISO} from '@/helpers/time/formatDate'

const props = defineProps<{
	task: Task,
	canMarkAsDone: boolean,
	canEditStatus: boolean,
	// The card's own label is its title, so the row leaves it out.
	hiddenLabelId?: number,
}>()

const {t} = useI18n({useScope: 'global'})
const router = useRouter()
const route = useRoute()
const {now} = useGlobalNow()

const task = computed(() => props.task)
const {markAsDone, isSaving} = useMarkTaskDone(task)
const {progress} = useTaskStatus(task, false)
const isLoading = useDelayedLoading(isSaving)

// Opens as a modal over the board, like Kanban, so the board keeps its scroll position.
const taskRoute = computed(() => ({name: 'task.detail', params: {id: props.task.id}, state: {backdropView: route.fullPath}}))
const labels = computed(() => (props.task.labels ?? []).filter(label => label.id !== props.hiddenLabelId))

// Only what needs attention: lower priorities are the noise this row exists to drop.
const priorityName = computed(() => {
	switch (props.task.priority) {
		case PRIORITIES.HIGH: return t('task.priority.high')
		case PRIORITIES.URGENT: return t('task.priority.urgent')
		case PRIORITIES.DO_NOW: return t('task.priority.doNow')
		default: return ''
	}
})

const dueDate = computed(() => {
	const due = props.task.due_date
	return due && new Date(due).getTime() > 0 ? due : null
})
const dueDateText = computed(() => {
	void now.value // Relative dates ("tomorrow") must move on with the clock.
	return dueDate.value ? formatDisplayDate(dueDate.value) : ''
})
const isOverdue = computed(() => dueDate.value !== null && new Date(dueDate.value).getTime() <= now.value.getTime())

function openTaskDetail(e: MouseEvent) {
	const target = e.target as HTMLElement
	const selected = window.getSelection()?.toString()
	// A bare newline is what a plain click leaves selected in some browsers, the same rule as the list row.
	if (target.closest('a, button, label, input, select, [role="button"]') || (selected && selected !== '\n')) {
		return
	}
	router.push(taskRoute.value)
}
</script>

<style lang="scss" scoped>
.board-task {
	display: flex;
	align-items: flex-start;
	// Wider than the checkbox's 44px hit area spills out (13px), so a click on the title's first letter opens it.
	gap: .875rem;
	padding: .5rem;
	border-radius: $radius;
	cursor: pointer;
	transition: background-color $transition;

	&:hover {
		background-color: var(--grey-100);
	}

	&:has(:focus-visible) {
		@include focus-ring;
	}

	&.loader-container.is-loading::after {
		inset-block-start: calc(50% - 1rem);
		inset-inline-start: calc(50% - 1rem);
		inline-size: 2rem;
		block-size: 2rem;
	}
}

.board-task__check {
	display: inline-flex;
	// Puts the box on the title's first line.
	margin-block-start: 2px;

	:deep(.fancy-checkbox) {
		block-size: 18px;
		padding: 0;

		// Extend the hit target to >=44x44 without affecting layout (WCAG 2.5.5).
		.base-checkbox__label {
			position: relative;

			&::before {
				content: '';
				position: absolute;
				inset-block-start: 50%;
				inset-inline-start: 50%;
				min-block-size: 44px;
				min-inline-size: 44px;
				transform: translate(-50%, -50%);
			}
		}
	}
}

.board-task__body {
	flex: 1;
	min-inline-size: 0;
}

// Not line-clamped: the clamp would also cut off the dots and avatars that trail a long title.
.board-task__title-line {
	font-size: .9375rem;
	line-height: 1.4;
	overflow-wrap: anywhere;
}

.board-task__title {
	color: var(--text);

	&:hover {
		color: var(--grey-900);
	}
}

.board-task__dot,
.board-task__assignees {
	margin-inline-start: .375rem;
	vertical-align: middle;
}

.board-task__assignees {
	display: inline-flex;

	// The list shows names beside avatars on phones; after a title they would wrap onto a line of their own.
	:deep(.username) {
		display: none;
	}
}

.board-task__priority {
	margin-inline-end: .25rem;
	color: var(--danger-text);
	font-weight: 700;
}

.board-task__meta {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .25rem .5rem;
	margin-block-start: .25rem;
	color: var(--grey-500);
	font-size: .8rem;

	&:not(:has(*)) {
		display: none;
	}

	.is-overdue {
		color: var(--danger-text);
	}
}

.board-task__dot {
	display: inline-block;
	inline-size: .5rem;
	block-size: .5rem;
	border-radius: 50%;
	background-color: var(--grey-300);
}
</style>
