import {computed} from 'vue'

import {useCreateTaskRelationMutation} from '@/client/queries/taskMutations'
import {useQuickAddTask} from '@/composables/useQuickAddTask'

export function useCreateSubtask() {
	const quickAdd = useQuickAddTask()
	const createRelation = useCreateTaskRelationMutation()

	// Failures show their own error toast (contextMutationOptions). A failed link leaves the new task without a parent.
	// Even a rejection can follow a created task (a failed assignee write), so callers must not keep the title for a retry.
	async function createSubtask(parent: {id: number, project_id: number}, title: string) {
		const child = await quickAdd.createNewTask({title, project_id: parent.project_id}, {keepProject: true})
		try {
			await createRelation.mutateAsync({
				taskId: parent.id,
				other_task_id: child.id!,
				relation_kind: 'subtask',
			})
		} catch {
			// toast already shown
		}
		return child
	}

	return {
		createSubtask,
		isCreating: computed(() => quickAdd.isLoading.value || createRelation.isPending.value),
	}
}
