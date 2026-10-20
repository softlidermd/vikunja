import {captureClientRequestContext, isClientRequestContextCurrent} from '@/client/requestContext'

// Module-wide, so the table, the list and the status select of one task wait for each other.
const pendingSaves = new Map<number, Promise<unknown>>()

// Saves of one task run one after another: each response carries the whole task,
// so a late response to an earlier edit would put the old value of a newer edit back.
function queueTaskSave<T>(taskId: number, save: () => Promise<T>): Promise<T | undefined> {
	// The mutation captures its context only when it starts, so fence a queued save to the session that made the edit.
	const request = captureClientRequestContext()
	const result = (pendingSaves.get(taskId) ?? Promise.resolve())
		.then(() => isClientRequestContextCurrent(request) ? save() : undefined)
	const settled = result.catch(() => undefined)
	pendingSaves.set(taskId, settled)
	void settled.then(() => {
		if (pendingSaves.get(taskId) === settled) {
			pendingSaves.delete(taskId)
		}
	})
	return result
}

export function useTaskSaveQueue() {
	return {queueTaskSave}
}
