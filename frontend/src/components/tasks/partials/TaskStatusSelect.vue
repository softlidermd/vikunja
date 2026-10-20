<template>
	<span
		v-if="kanbanView && currentBucket"
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
				:aria-disabled="moveMutation.isPending.value || undefined"
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
import {useProjects} from '@/composables/useProjects'
import {getStatusKanbanView} from '@/helpers/projectView'
import {useBaseStore} from '@/stores/base'

const props = defineProps<{
	task: Task,
	canWrite: boolean,
}>()

const projectList = useProjects()
const baseStore = useBaseStore()
const moveMutation = useMoveTaskMutation()

const kanbanView = computed(() => getStatusKanbanView(
	projectList.projects[props.task.project_id ?? 0]?.views,
	baseStore.currentProjectViewId,
))

const bucketsResult = useQuery(computed(() => ({
	...bucketsQuery(props.task.project_id ?? 0, kanbanView.value?.id ?? 0),
	enabled: props.canWrite && (props.task.project_id ?? 0) > 0 && !!kanbanView.value?.id,
})))

const currentBucket = computed(() => props.task.buckets?.find(b => b.project_view_id === kanbanView.value?.id))
// Until the buckets load, the select still has to show the current one.
const options = computed(() => bucketsResult.data.value?.length
	? bucketsResult.data.value
	: [currentBucket.value!])

const selected = ref<number>()
watch(() => currentBucket.value?.id, id => {
	selected.value = id
}, {immediate: true})

async function changeBucket() {
	const view = kanbanView.value
	const bucket = selected.value
	// A second move while one is in flight could land first and be overwritten by the older response.
	if (moveMutation.isPending.value || !view?.id || bucket === undefined || bucket === currentBucket.value?.id) {
		selected.value = currentBucket.value?.id
		return
	}
	try {
		await moveMutation.mutateAsync({
			project: props.task.project_id!,
			view: view.id,
			bucket,
			task: props.task,
		})
	} catch {
		// The error toast is done by contextMutationOptions; put the select back unless the user picked again.
		if (selected.value === bucket) {
			selected.value = currentBucket.value?.id
		}
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

		&[aria-disabled] {
			cursor: progress;
			opacity: .6;
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
