import {computed, type Ref} from 'vue'
import {useI18n} from 'vue-i18n'

import type {Task} from '@/client/generated'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'
import {useTaskSaveQueue} from '@/composables/useTaskSaveQueue'
import {playPopSound} from '@/helpers/playPop'
import {success} from '@/message'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'

/**
 * The done checkbox of a task row: saves in order with the task's other edits,
 * plays the pop, and offers an undo toast.
 */
export function useMarkTaskDone(
	task: Ref<Task>,
	onSaved?: (updated: Task, reverted: boolean) => void,
) {
	const {t} = useI18n({useScope: 'global'})
	const updateTask = useUpdateTaskMutation(true)
	const {queueTaskSave} = useTaskSaveQueue()

	const isRepeating = computed(() => (task.value.repeat_after ?? 0) > 0
		|| ((task.value.repeat_after ?? 0) === 0
			&& task.value.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH))

	let oldTask: Task

	async function markAsDone(checked: boolean, wasReverted: boolean = false) {
		if (!wasReverted) oldTask = {...task.value}

		// Queue the request now, with the intended done value snapshotted, so a re-render or
		// teardown during the animation delay can neither drop the save nor make it send a stale state.
		const source = wasReverted && isRepeating.value ? oldTask : task.value
		const update = {
			...source,
			id: source.id!,
			done: checked,
		}
		const updatePromise = queueTaskSave(update.id, () => updateTask.mutateAsync(update)).catch(() => undefined)

		const finish = async () => {
			const newTask = await updatePromise
			if (!newTask) return

			onSaved?.(newTask, wasReverted)
			if (wasReverted) {
				return
			}

			if (checked) {
				playPopSound()
			}

			let message = t('task.doneSuccess')
			if (!task.value.done && !isRepeating.value) {
				message = t('task.undoneSuccess')
			}

			success({message}, [{
				title: t('task.undo'),
				callback: () => markAsDone(!checked, true),
			}])
		}

		if (checked) {
			setTimeout(finish, 300) // Delay only the follow-up to show the animation when marking a task as done
		} else {
			await finish() // Don't delay it when un-marking it as it doesn't have an animation the other way around
		}
	}

	return {markAsDone, isRepeating, isSaving: updateTask.isPending}
}
