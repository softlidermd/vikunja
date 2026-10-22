import {describe, it, expect, vi, beforeEach} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia, setActivePinia} from 'pinia'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'
import {createRouter, createMemoryHistory} from 'vue-router'

import type {Bucket, Task} from '@/client/generated'
import {bucketKeys} from '@/client/queries/kanban'
import {useTaskSaveQueue} from '@/composables/useTaskSaveQueue'

const sdk = vi.hoisted(() => ({
	taskBucketUpdate: vi.fn(),
	bucketsList: vi.fn(),
}))

const project = vi.hoisted(() => ({
	id: 1,
	views: [
		{
			id: 3,
			view_kind: 'list',
		},
		{
			id: 4,
			view_kind: 'kanban',
			bucket_configuration_mode: 'manual',
			default_bucket_id: 10,
			done_bucket_id: 12,
		},
	],
}))

vi.mock('@/client/generated', () => sdk)
vi.mock('@/message', () => ({
	error: vi.fn(),
	success: vi.fn(),
}))
vi.mock('@/composables/useProjects', () => ({useProjects: () => ({projects: {1: project}})}))
vi.mock('vue-i18n', async importOriginal => ({
	...(await importOriginal<typeof import('vue-i18n')>()),
	useI18n: () => ({t: (key: string) => key}),
}))

import TaskStatusSelect from './TaskStatusSelect.vue'

const buckets: Bucket[] = [
	{
		id: 10,
		title: 'To-Do',
		project_view_id: 4,
	},
	{
		id: 11,
		title: 'Doing',
		project_view_id: 4,
	},
	{
		id: 12,
		title: 'Done',
		project_view_id: 4,
	},
]

const task = {
	id: 7,
	project_id: 1,
	title: 'Write the invoice',
	done: false,
	buckets: [buckets[0]],
} as Task

interface ExtraProps {
	task?: Task
	hideDefaultBucket?: boolean
}

function mountSelect(canWrite = true, extraProps: ExtraProps = {}) {
	const queryClient = new QueryClient({defaultOptions: {queries: {retry: false}}})
	queryClient.setQueryData(bucketKeys.list(1, 4), buckets)
	return mount(TaskStatusSelect, {
		props: {
			task,
			canWrite,
			...extraProps,
		},
		global: {
			plugins: [
				createPinia(),
				createRouter({
					history: createMemoryHistory(),
					routes: [{
						path: '/',
						component: {render: () => null},
					}],
				}),
				[VueQueryPlugin, {queryClient}],
			],
			mocks: {$t: (key: string) => key},
			stubs: {Icon: true},
		},
	})
}

describe('TaskStatusSelect', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		sdk.taskBucketUpdate.mockReset()
		sdk.bucketsList.mockReset()
		sdk.bucketsList.mockResolvedValue({data: {items: buckets}})
	})

	it('moves the task through the kanban bucket endpoint of the project\'s kanban view', async () => {
		sdk.taskBucketUpdate.mockResolvedValue({
			data: {
				bucket_id: 12,
				task: {
					...task,
					done: true,
				},
			},
		})
		const wrapper = mountSelect()
		await flushPromises()

		expect(wrapper.get('select').element.value).toBe('10')
		await wrapper.get('select').setValue('12')
		await flushPromises()

		expect(sdk.taskBucketUpdate).toHaveBeenCalledOnce()
		expect(sdk.taskBucketUpdate.mock.calls[0][0]).toMatchObject({
			path: {
				project: 1,
				view: 4,
				bucket: 12,
			},
			body: {task_id: 7},
		})
	})

	it('hides the default bucket only when asked to', async () => {
		const inDoing = {...task, buckets: [buckets[1]]}
		expect(mountSelect(true).find('.task-status').exists()).toBe(true)
		expect(mountSelect(true, {hideDefaultBucket: true}).find('.task-status').exists()).toBe(false)
		expect(mountSelect(true, {hideDefaultBucket: true, task: inDoing}).find('.task-status').exists()).toBe(true)
	})

	it('treats the leftmost bucket as the default once the default bucket was deleted', async () => {
		const view = project.views[1] as {default_bucket_id?: number}
		view.default_bucket_id = 0
		try {
			const wrapper = mountSelect(true, {hideDefaultBucket: true})
			await flushPromises()
			expect(wrapper.find('.task-status').exists()).toBe(false)
		} finally {
			view.default_bucket_id = 10
		}
	})

	it('puts the old bucket back when the move fails', async () => {
		sdk.taskBucketUpdate.mockRejectedValue({
			status: 412,
			detail: 'limit exceeded',
		})
		const wrapper = mountSelect()
		await flushPromises()

		await wrapper.get('select').setValue('11')
		await flushPromises()

		expect(sdk.taskBucketUpdate).toHaveBeenCalledOnce()
		expect(wrapper.get('select').element.value).toBe('10')
	})

	it('queues a second choice made while the first one is saving', async () => {
		let resolveFirst!: () => void
		sdk.taskBucketUpdate
			.mockImplementationOnce(() => new Promise(resolve => {
				resolveFirst = () => resolve({data: {bucket_id: 11}})
			}))
			.mockResolvedValueOnce({data: {bucket_id: 12}})
		const wrapper = mountSelect()
		await flushPromises()

		await wrapper.get('select').setValue('11')
		await wrapper.get('select').setValue('12')
		await flushPromises()
		expect(sdk.taskBucketUpdate).toHaveBeenCalledOnce()

		resolveFirst()
		await flushPromises()
		expect(sdk.taskBucketUpdate).toHaveBeenCalledTimes(2)
		expect(sdk.taskBucketUpdate.mock.calls[1][0]).toMatchObject({path: {bucket: 12}})
		expect(wrapper.get('select').element.value).toBe('12')
	})

	it('waits for another save of the same task before moving it', async () => {
		let resolveSave!: () => void
		const {queueTaskSave} = useTaskSaveQueue()
		void queueTaskSave(task.id!, () => new Promise<void>(resolve => {
			resolveSave = resolve
		}))
		sdk.taskBucketUpdate.mockResolvedValue({data: {bucket_id: 11}})
		const wrapper = mountSelect()
		await flushPromises()

		await wrapper.get('select').setValue('11')
		await flushPromises()
		expect(sdk.taskBucketUpdate).not.toHaveBeenCalled()

		resolveSave()
		await flushPromises()
		expect(sdk.taskBucketUpdate).toHaveBeenCalledOnce()
	})

	it('goes back to the bucket a repeating task was reopened in', async () => {
		sdk.taskBucketUpdate.mockResolvedValue({
			data: {
				bucket_id: 10,
				task: {
					...task,
					repeat_after: 86400,
				},
			},
		})
		const wrapper = mountSelect()
		await flushPromises()

		await wrapper.get('select').setValue('12')
		await flushPromises()

		expect(sdk.taskBucketUpdate).toHaveBeenCalledOnce()
		expect(wrapper.get('select').element.value).toBe('10')
	})

	it('shows the bucket as text to readers', async () => {
		const wrapper = mountSelect(false)
		await flushPromises()

		expect(wrapper.find('select').exists()).toBe(false)
		expect(wrapper.text()).toBe('To-Do')
	})
})
