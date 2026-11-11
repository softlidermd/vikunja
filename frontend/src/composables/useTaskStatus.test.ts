import {defineComponent, h} from 'vue'
import {flushPromises, mount} from '@vue/test-utils'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'
import {describe, it, expect, onTestFinished, vi} from 'vitest'

import type {Bucket, ProjectView, Task} from '@/client/generated'
import {getTaskProgress, useTaskStatus} from './useTaskStatus'

const sdk = vi.hoisted(() => ({bucketsList: vi.fn()}))
vi.mock('@/client/generated', () => sdk)
vi.mock('@/client/queries/taskMutations', () => ({useMoveTaskMutation: () => ({})}))
vi.mock('@/composables/useTaskSaveQueue', () => ({useTaskSaveQueue: () => ({queueTaskSave: vi.fn()})}))
vi.mock('@/stores/base', () => ({useBaseStore: () => ({currentProjectViewId: undefined})}))
vi.mock('@/composables/useProjects', () => ({
	useProjects: () => ({
		projects: {
			1: {
				views: [{
					id: 4,
					view_kind: 'kanban',
					bucket_configuration_mode: 'manual',
					default_bucket_id: 9,
					done_bucket_id: 12,
				}],
			},
		},
	}),
}))

const view = {
	id: 4,
	default_bucket_id: 10,
	done_bucket_id: 12,
} as ProjectView
const buckets: Bucket[] = [
	{id: 10, title: 'To-Do'},
	{id: 11, title: 'Doing'},
	{id: 12, title: 'Done'},
]
const ideaView = {...view, default_bucket_id: 9}
const ideaBuckets: Bucket[] = [{id: 9, title: 'Idea'}, ...buckets]

function inBucket(bucket: number, done = false) {
	return {
		id: 1,
		project_id: 1,
		done,
		buckets: [{id: bucket, project_view_id: 4}],
	} as Task
}

describe('getTaskProgress', () => {
	it('tells the default, an in-between and the done bucket apart', () => {
		expect(getTaskProgress(inBucket(10), view, buckets)).toBeUndefined()
		expect(getTaskProgress(inBucket(11), view, buckets)).toBe('in-progress')
		expect(getTaskProgress(inBucket(12), view, buckets)).toBe('done')
	})

	it('keeps Idea and the planned To-Do out of progress on an Idea board', () => {
		expect(getTaskProgress(inBucket(9), ideaView, ideaBuckets)).toBeUndefined()
		expect(getTaskProgress(inBucket(10), ideaView, ideaBuckets)).toBeUndefined()
		expect(getTaskProgress(inBucket(11), ideaView, ideaBuckets)).toBe('in-progress')
		expect(getTaskProgress(inBucket(12), ideaView, ideaBuckets)).toBe('done')
	})

	it('shows a done task as done even without a Kanban view', () => {
		expect(getTaskProgress({id: 1, done: true} as Task, null, [])).toBe('done')
	})

	it('marks nothing in progress while the buckets are unknown', () => {
		expect(getTaskProgress(inBucket(11), view, [])).toBeUndefined()
	})
})

function withQueryClient<T>(setup: () => T): T {
	let result!: T
	const queryClient = new QueryClient()
	const wrapper = mount(defineComponent({
		setup() {
			result = setup()
			return () => h('div')
		},
	}), {global: {plugins: [[VueQueryPlugin, {queryClient}]]}})
	onTestFinished(() => {
		wrapper.unmount()
		queryClient.clear()
	})
	return result
}

describe('useTaskStatus', () => {
	it('lists Idea, To-Do, Doing and Done for an editable status select on an Idea board', async () => {
		sdk.bucketsList.mockResolvedValue({data: {items: ideaBuckets}})
		const status = withQueryClient(() => useTaskStatus(inBucket(9), true))
		await flushPromises()
		expect(status.buckets.value.map(bucket => bucket.title)).toEqual(['Idea', 'To-Do', 'Doing', 'Done'])
		expect(status.progress.value).toBeUndefined()
	})

	it('needs no bucket list for a read-only task in the default bucket', async () => {
		sdk.bucketsList.mockClear()
		withQueryClient(() => useTaskStatus(inBucket(9), false))
		await flushPromises()
		expect(sdk.bucketsList).not.toHaveBeenCalled()
	})

	it('loads the bucket list for a read-only task outside the default and done buckets', async () => {
		sdk.bucketsList.mockClear()
		sdk.bucketsList.mockResolvedValue({data: {items: ideaBuckets}})
		const status = withQueryClient(() => useTaskStatus(inBucket(11), false))
		await flushPromises()
		expect(sdk.bucketsList).toHaveBeenCalledTimes(1)
		expect(status.progress.value).toBe('in-progress')
	})
})
