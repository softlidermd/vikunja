import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import type {ProjectView, Task} from '@/client/generated'
import {bucketsQuery} from '@/client/queries/kanban'
import {useMoveTaskMutation} from '@/client/queries/taskMutations'
import {useProjects} from '@/composables/useProjects'
import {useTaskSaveQueue} from '@/composables/useTaskSaveQueue'
import {getStatusKanbanView} from '@/helpers/projectView'
import {getDefaultBucketId} from '@/helpers/task'
import {useBaseStore} from '@/stores/base'

export type TaskProgress = 'in-progress' | 'done' | undefined

// Bucket names are free text, so "in progress" is any bucket besides the default and the done one.
export function getTaskProgress(
	task: Task,
	view: ProjectView | null | undefined,
	defaultBucketId: number | undefined,
): TaskProgress {
	const bucket = task.buckets?.find(b => b.project_view_id === view?.id)
	if (task.done || (bucket && bucket.id === view?.done_bucket_id)) {
		return 'done'
	}
	return bucket && defaultBucketId && bucket.id !== defaultBucketId ? 'in-progress' : undefined
}

// A task's status is its bucket in the project's Kanban view (see getStatusKanbanView).
export function useTaskStatus(task: MaybeRefOrGetter<Task>, loadBuckets: MaybeRefOrGetter<boolean>) {
	const projectList = useProjects()
	const baseStore = useBaseStore()
	const moveMutation = useMoveTaskMutation()
	const {queueTaskSave} = useTaskSaveQueue()

	const kanbanView = computed(() => getStatusKanbanView(
		projectList.projects[toValue(task).project_id ?? 0]?.views,
		baseStore.currentProjectViewId,
	))

	const bucketsResult = useQuery(computed(() => {
		const options = bucketsQuery(toValue(task).project_id ?? 0, kanbanView.value?.id ?? 0)
		return {
			...options,
			// Deleting the default bucket leaves its id at 0; then the leftmost bucket is the default.
			enabled: (toValue(loadBuckets) || !kanbanView.value?.default_bucket_id) && options.enabled,
		}
	}))
	const buckets = computed(() => bucketsResult.data.value ?? [])

	const currentBucket = computed(() => toValue(task).buckets
		?.find(b => b.project_view_id === kanbanView.value?.id))
	const defaultBucketId = computed(() => kanbanView.value
		? getDefaultBucketId(kanbanView.value, buckets.value)
		: undefined)
	const progress = computed(() => getTaskProgress(toValue(task), kanbanView.value, defaultBucketId.value))

	// Resolves to the bucket the task really landed in: a repeating task moved to done is reopened in its old one.
	async function moveToBucket(bucket: number) {
		const view = kanbanView.value?.id
		const current = toValue(task)
		if (view === undefined || current.id === undefined) {
			return undefined
		}
		// Queued with the task's other saves: the response is the whole task and would overwrite a newer edit.
		const result = await queueTaskSave(current.id, () => moveMutation.mutateAsync({
			project: current.project_id!,
			view,
			bucket,
			task: current,
		}))
		return result?.bucket_id
	}

	return {
		kanbanView,
		buckets,
		currentBucket,
		defaultBucketId,
		progress,
		moveToBucket,
	}
}
