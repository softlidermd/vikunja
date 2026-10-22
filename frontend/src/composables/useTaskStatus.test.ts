import {describe, it, expect} from 'vitest'

import type {ProjectView, Task} from '@/client/generated'
import {getDefaultBucketId, getTaskProgress} from './useTaskStatus'

const view = {
	id: 4,
	default_bucket_id: 10,
	done_bucket_id: 12,
} as ProjectView

function inBucket(bucket: number, done = false) {
	return {
		id: 1,
		done,
		buckets: [{id: bucket, project_view_id: 4}],
	} as Task
}

describe('getTaskProgress', () => {
	it('tells the default, an in-between and the done bucket apart', () => {
		expect(getTaskProgress(inBucket(10), view, 10)).toBeUndefined()
		expect(getTaskProgress(inBucket(11), view, 10)).toBe('in-progress')
		expect(getTaskProgress(inBucket(12), view, 10)).toBe('done')
	})

	it('shows a done task as done even without a Kanban view', () => {
		expect(getTaskProgress({id: 1, done: true} as Task, null, undefined)).toBe('done')
	})

	it('marks nothing in progress while the default bucket is unknown', () => {
		expect(getTaskProgress(inBucket(11), view, undefined)).toBeUndefined()
	})
})

describe('getDefaultBucketId', () => {
	it('falls back to the leftmost bucket once the default bucket was deleted', () => {
		const buckets = [{id: 21, position: 2}, {id: 20, position: 1}]
		expect(getDefaultBucketId({...view, default_bucket_id: 0}, buckets)).toBe(20)
	})
})
