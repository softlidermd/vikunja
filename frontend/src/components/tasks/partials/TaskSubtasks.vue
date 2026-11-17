<template>
	<div
		v-if="subtasks.length > 0 || canWrite"
		class="content details task-subtasks"
	>
		<h2
			v-if="subtasks.length > 0 || adding"
			class="task-section-title"
		>
			{{ $t('task.subtasks.title', {done: doneCount, total: subtasks.length}) }}
		</h2>
		<ul
			v-if="subtasks.length > 0"
			class="task-subtasks__list"
		>
			<li
				v-for="subtask in subtasks"
				:key="subtask.id"
				class="task-subtasks__item"
			>
				<FancyCheckbox
					:model-value="subtask.done ?? false"
					:disabled="!canWrite"
					:aria-label="$t('task.detail.markAsDone', {task: subtask.title})"
					@update:modelValue="done => toggleDone(subtask, done)"
				/>
				<RouterLink
					:to="{name: route.name as string, params: {id: subtask.id}, state: {backdropView: route.fullPath}}"
					:class="{'is-strikethrough has-text-grey': subtask.done}"
				>
					{{ subtask.title }}
				</RouterLink>
			</li>
		</ul>
		<div
			v-if="adding"
			class="task-subtasks__new"
		>
			<input
				v-model="title"
				v-focus.always
				class="input"
				:placeholder="$t('task.relation.subtaskPlaceholder')"
				:aria-label="$t('task.relation.subtaskPlaceholder')"
				:readonly="isCreating"
				@keydown.enter.exact="e => !e.isComposing && add()"
				@keydown.esc.prevent.stop="close()"
			>
			<p class="help">
				{{ $t('task.subtasks.hint') }}
			</p>
		</div>
		<BaseButton
			v-else-if="canWrite"
			class="task-subtasks__add d-print-none"
			@click="adding = true"
		>
			<Icon icon="plus" />
			{{ $t('task.subtasks.add') }}
		</BaseButton>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useRoute} from 'vue-router'

import type {Task} from '@/client/generated'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'
import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import {useCreateSubtask} from '@/composables/useCreateSubtask'
import {playPopSound} from '@/helpers/playPop'

const props = defineProps<{
	task: Task,
	canWrite: boolean,
}>()

const route = useRoute()
const updateTask = useUpdateTaskMutation()
const {submitSubtask, isCreating} = useCreateSubtask()

const subtasks = computed(() => props.task.related_tasks?.subtask ?? [])
const doneCount = computed(() => subtasks.value.filter(subtask => subtask.done).length)

const adding = ref(false)
const title = ref('')

function close() {
	adding.value = false
	title.value = ''
}

function add() {
	if (props.task.id === undefined || props.task.project_id === undefined) {
		return
	}
	// The field stays open (readonly while saving keeps the focus) for the next subtask.
	return submitSubtask({id: props.task.id, project_id: props.task.project_id}, title)
}

async function toggleDone(subtask: Task, done: boolean) {
	try {
		await updateTask.mutateAsync({...subtask, id: subtask.id!, done})
	} catch {
		// Failed mutations already show an error toast (contextMutationOptions).
		return
	}
	if (done) {
		playPopSound()
	}
}
</script>

<style lang="scss" scoped>
.task-subtasks__list {
	margin: 0 0 .5rem;
	padding: 0;
	list-style: none;
}

.task-subtasks__item {
	display: flex;
	align-items: center;
	gap: .5rem;
	padding: .25rem .5rem;
	border-radius: $radius;

	&:hover {
		background-color: var(--grey-100);
	}

	a {
		color: var(--text);
	}

	// Its padding-block-start would push the box below the title's centre line.
	.fancy-checkbox {
		padding: 0;
	}
}

.task-subtasks__new .help {
	margin-block-start: .25rem;
	color: var(--text-muted);
}

.task-subtasks__add {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	padding: .375rem .75rem;
	border: 1px dashed var(--grey-300);
	border-radius: $radius;
	font-size: .875rem;
	font-weight: 600;
	color: var(--link);

	&:hover {
		border-color: var(--link);
	}
}
</style>
