import {describe, expect, it} from 'vitest'
import {nestTaskRows, type NestableTask} from '@/helpers/nestTaskRows'

function task(id: number, subtasks: number[] = [], parents: number[] = []): NestableTask {
	return {
		id,
		related_tasks: {
			subtask: subtasks.map(i => ({id: i})),
			parenttask: parents.map(i => ({id: i})),
		},
	}
}

const shape = (rows: ReturnType<typeof nestTaskRows>) => rows.map(r => [r.task.id, r.depth, r.hasChildren])

describe('nestTaskRows', () => {
	it('keeps a flat page as it is', () => {
		expect(shape(nestTaskRows([task(1), task(2)]))).toEqual([[1, 0, false], [2, 0, false]])
	})

	it('puts subtasks under their parent in page order, at any depth', () => {
		const tasks = [task(4, [], [2]), task(1, [2, 3]), task(3, [], [1]), task(2, [4], [1])]
		expect(shape(nestTaskRows(tasks))).toEqual([
			[1, 0, true],
			[3, 1, false],
			[2, 1, true],
			[4, 2, false],
		])
	})

	it('tells each row where the tree lines run on past it', () => {
		// 1 ─┬ 2 ─┬ 4
		//    │    └ 5
		//    └ 3
		const tasks = [task(1, [2, 3]), task(2, [4, 5], [1]), task(3, [], [1]), task(4, [], [2]), task(5, [], [2])]
		expect(nestTaskRows(tasks).map(r => [r.task.id, r.continues])).toEqual([
			[1, []],
			[2, [true]],
			[4, [true, true]],
			[5, [true, false]],
			[3, [false]],
		])
	})

	it('keeps a subtask whose parent is not on the page as a top-level row', () => {
		expect(shape(nestTaskRows([task(5, [], [99])]))).toEqual([[5, 0, false]])
	})

	it('shows a subtask with two parents once, under the first parent', () => {
		const tasks = [task(1, [3]), task(2, [3]), task(3, [], [1, 2])]
		expect(shape(nestTaskRows(tasks))).toEqual([[1, 0, true], [3, 1, false], [2, 0, false]])
	})

	it('leaves out the descendants of a collapsed task', () => {
		const tasks = [task(1, [2]), task(2, [3], [1]), task(3, [], [2]), task(4)]
		expect(shape(nestTaskRows(tasks, new Set([1])))).toEqual([[1, 0, true], [4, 0, false]])
	})

	it('does not drop tasks that form a cycle', () => {
		const tasks = [task(1, [2], [2]), task(2, [1], [1])]
		expect(nestTaskRows(tasks).map(r => r.task.id).sort()).toEqual([1, 2])
	})
})
