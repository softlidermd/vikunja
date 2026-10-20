import {describe, it, expect, vi, beforeEach} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia, setActivePinia} from 'pinia'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'
import {createRouter, createMemoryHistory} from 'vue-router'

import type {Bucket, Task} from '@/client/generated'
import {bucketKeys} from '@/client/queries/kanban'

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

function mountSelect(canWrite = true) {
	const queryClient = new QueryClient({defaultOptions: {queries: {retry: false}}})
	queryClient.setQueryData(bucketKeys.list(1, 4), buckets)
	return mount(TaskStatusSelect, {
		props: {
			task,
			canWrite,
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

	it('shows the bucket as text to readers', async () => {
		const wrapper = mountSelect(false)
		await flushPromises()

		expect(wrapper.find('select').exists()).toBe(false)
		expect(wrapper.text()).toBe('To-Do')
	})
})
