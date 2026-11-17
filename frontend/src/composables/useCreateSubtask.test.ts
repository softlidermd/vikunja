import {describe, it, expect, vi, beforeEach} from 'vitest'
import {ref} from 'vue'

const createNewTask = vi.fn()
const createRelation = vi.fn()

vi.mock('@/composables/useQuickAddTask', () => ({
	useQuickAddTask: () => ({createNewTask, isLoading: ref(false)}),
}))
vi.mock('@/client/queries/taskMutations', () => ({
	useCreateTaskRelationMutation: () => ({mutateAsync: createRelation, isPending: ref(false)}),
}))

import {useCreateSubtask} from './useCreateSubtask'

describe('useCreateSubtask', () => {
	beforeEach(() => {
		vi.clearAllMocks()
		createNewTask.mockResolvedValue({id: 42, title: 'child', project_id: 3})
		createRelation.mockResolvedValue({})
	})

	it('creates the task in the parent project and links it as a subtask of the parent', async () => {
		const child = await useCreateSubtask().createSubtask({id: 7, project_id: 3}, 'child')
		expect(createNewTask).toHaveBeenCalledWith({title: 'child', project_id: 3}, {keepProject: true})
		expect(createRelation).toHaveBeenCalledWith({taskId: 7, other_task_id: 42, relation_kind: 'subtask'})
		expect(child.id).toBe(42)
	})

	it('resolves when only the link fails, so a retry does not create the task twice', async () => {
		createRelation.mockRejectedValue(new Error('link failed'))
		await expect(useCreateSubtask().createSubtask({id: 7, project_id: 3}, 'child')).resolves.toMatchObject({id: 42})
	})

	it('rejects without linking when the task could not be created', async () => {
		createNewTask.mockRejectedValue(new Error('create failed'))
		await expect(useCreateSubtask().createSubtask({id: 7, project_id: 3}, 'child')).rejects.toThrow('create failed')
		expect(createRelation).not.toHaveBeenCalled()
	})
})
