<template>
	<span
		v-if="kanbanView && currentBucket && !isHiddenDefault"
		class="task-status"
		:class="{
			'is-done': currentBucket.id === kanbanView.done_bucket_id,
			'is-editable': canWrite,
		}"
	>
		<template v-if="canWrite">
			<select
				v-model="selected"
				:aria-label="$t('task.detail.bucketSelectLabel', {bucket: currentBucket.title})"
				@click.stop
				@change="changeBucket"
			>
				<option
					v-for="bucket in options"
					:key="bucket.id"
					:value="bucket.id"
				>
					{{ bucket.title }}
				</option>
			</select>
			<Icon
				icon="chevron-down"
				class="task-status__caret"
			/>
		</template>
		<template v-else>
			{{ currentBucket.title }}
		</template>
	</span>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {bucketsQuery} from '@/client/queries/kanban'
import {useMoveTaskMutation} from '@/client/queries/taskMutations'
import {isRequestContextAbort} from '@/client/requestContext'
import {useProjects} from '@/composables/useProjects'
import {useTaskSaveQueue} from '@/composables/useTaskSaveQueue'
import {getStatusKanbanView} from '@/helpers/projectView'
import {useBaseStore} from '@/stores/base'

const props = defineProps<{
	task: Task,
	canWrite: boolean,
	// Where most tasks sit, so a compact row only shows the other statuses.
	hideDefaultBucket?: boolean,
}>()

const projectList = useProjects()
const baseStore = useBaseStore()
const moveMutation = useMoveTaskMutation()
const {queueTaskSave} = useTaskSaveQueue()

const kanbanView = computed(() => getStatusKanbanView(
	projectList.projects[props.task.project_id ?? 0]?.views,
	baseStore.currentProjectViewId,
))

const bucketsResult = useQuery(computed(() => {
	const options = bucketsQuery(props.task.project_id ?? 0, kanbanView.value?.id ?? 0)
	return {
		...options,
		enabled: props.canWrite && options.enabled,
	}
}))

const currentBucket = computed(() => props.task.buckets?.find(b => b.project_view_id === kanbanView.value?.id))
// Deleting the default bucket leaves the id at 0, and the server then uses the leftmost bucket.
const defaultBucketId = computed(() => kanbanView.value?.default_bucket_id ||
	[...(bucketsResult.data.value ?? [])].sort((a, b) => (a.position ?? 0) - (b.position ?? 0))[0]?.id)
const isHiddenDefault = computed(() => props.hideDefaultBucket && currentBucket.value?.id === defaultBucketId.value)
// Until the buckets load, the select still has to show the current one.
const options = computed(() => bucketsResult.data.value?.length
	? bucketsResult.data.value
	: [currentBucket.value!])

const selected = ref<number>()
watch(() => currentBucket.value?.id, id => {
	selected.value = id
}, {immediate: true})

async function changeBucket() {
	const view = kanbanView.value?.id
	const bucket = selected.value
	const task = props.task
	if (view === undefined || bucket === undefined || task.id === undefined) {
		return
	}
	let placed: number | undefined
	try {
		// Queued with the task's other saves: the response is the whole task and would overwrite a newer edit.
		const result = await queueTaskSave(task.id, () => moveMutation.mutateAsync({
			project: task.project_id!,
			view,
			bucket,
			task,
		}))
		// A repeating task moved to done is reopened in its old bucket, so the watched bucket id does not change.
		placed = result?.bucket_id
	} catch (e) {
		if (isRequestContextAbort(e)) {
			return
		}
		// The error toast is done by contextMutationOptions.
	}
	// Show where the task really is, unless the user already picked again.
	if (selected.value === bucket) {
		selected.value = placed ?? currentBucket.value?.id
	}
}
</script>

<style lang="scss" scoped>
.task-status {
	position: relative;
	display: inline-flex;
	align-items: center;
	max-inline-size: 12rem;
	padding: .0625rem .625rem;
	border-radius: 999px;
	background: var(--grey-100);
	color: var(--grey-700);
	font-size: .8rem;
	line-height: 1.375rem;
	white-space: nowrap;
	vertical-align: middle;

	&.is-done {
		// A tint with theme text: bright green text on a light background fails contrast.
		background: hsla(var(--success-h), var(--success-s), var(--success-l), .2);
		color: var(--text);
	}

	&.is-editable {
		padding: 0;

		&:hover {
			background: var(--grey-200);
		}

		&.is-done:hover {
			background: hsla(var(--success-h), var(--success-s), var(--success-l), .3);
		}
	}

	select {
		appearance: none;
		max-inline-size: 100%;
		padding-block: .125rem;
		padding-inline: .625rem 1.5rem;
		border: 0;
		border-radius: inherit;
		background: transparent;
		color: inherit;
		font: inherit;
		text-overflow: ellipsis;
		cursor: pointer;

		&:focus-visible {
			outline: 2px solid hsla(var(--primary-hsl), .5);
			outline-offset: 1px;
		}
	}

	&__caret {
		position: absolute;
		inset-inline-end: .625rem;
		font-size: .55rem;
		pointer-events: none;
	}
}
</style>
