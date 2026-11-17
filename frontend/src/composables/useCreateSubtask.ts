import {computed, ref, type Ref} from 'vue'

import {useCreateTaskRelationMutation} from '@/client/queries/taskMutations'
import {useQuickAddTask} from '@/composables/useQuickAddTask'

export function useCreateSubtask() {
	const quickAdd = useQuickAddTask()
	const createRelation = useCreateTaskRelationMutation()
	// Set before the first await: the mutations only start after the quick-add lookups (projects, assignees).
	const submitting = ref(false)
	const isCreating = computed(() => submitting.value || quickAdd.isLoading.value || createRelation.isPending.value)

	// Failed writes show their own error toast (contextMutationOptions).
	// A failed link leaves the new task without a parent.
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

	// Returns false when there was nothing to submit. The title is cleared after every attempt: even a
	// rejection can follow a created task (a failed assignee write), so a retry could duplicate it.
	async function submitSubtask(parent: {id: number, project_id: number}, title: Ref<string>) {
		const text = title.value.trim()
		if (text === '' || isCreating.value) {
			return false
		}
		submitting.value = true
		try {
			await createSubtask(parent, text)
		} catch {
			// toast already shown
		} finally {
			title.value = ''
			submitting.value = false
		}
		return true
	}

	return {createSubtask, submitSubtask, isCreating}
}
