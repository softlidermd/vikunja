// Estimated effort in hours; the api accepts only these values.
export const TASK_SIZE_LONG = 999

export const TASK_SIZES = [0, 1, 4, 8, 20, 40, TASK_SIZE_LONG] as const

export type TaskSize = typeof TASK_SIZES[number]

const SIZE_LABEL_KEYS: Record<TaskSize, string> = {
	0: 'task.size.unset',
	1: 'task.size.hour',
	4: 'task.size.halfDay',
	8: 'task.size.day',
	20: 'task.size.days',
	40: 'task.size.week',
	[TASK_SIZE_LONG]: 'task.size.long',
}

export function taskSizeLabelKey(size: number) {
	return SIZE_LABEL_KEYS[size as TaskSize] ?? SIZE_LABEL_KEYS[0]
}
