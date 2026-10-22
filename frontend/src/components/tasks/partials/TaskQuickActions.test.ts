import {describe, it, expect, vi, beforeEach} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia} from 'pinia'
import {createRouter, createMemoryHistory} from 'vue-router'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'

const sdk = vi.hoisted(() => ({
	taskAssigneesCreate: vi.fn(async () => ({data: {}})),
	taskAssigneesDelete: vi.fn(async () => ({data: {}})),
}))

vi.mock('@/client/generated', () => sdk)
vi.mock('@/message', () => ({error: vi.fn(), success: vi.fn()}))
vi.mock('@/composables/useProjects', () => ({useProjects: () => ({projects: {}})}))
vi.mock('@/stores/auth', () => ({
	useAuthStore: () => ({
		info: {
			id: 5,
			username: 'me',
		},
		isLinkShareAuth: false,
	}),
}))
vi.mock('vue-i18n', async importOriginal => ({
	...(await importOriginal<typeof import('vue-i18n')>()),
	useI18n: () => ({t: (key: string) => key}),
}))

import TaskQuickActions from './TaskQuickActions.vue'

function mountMenu(assignees: {id: number}[]) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{
				path: '/tasks/:id',
				name: 'task.detail',
				component: {render: () => null},
			},
		],
	})
	return mount(TaskQuickActions, {
		props: {
			task: {
				id: 7,
				project_id: 1,
				title: 'Write the invoice',
				assignees,
			} as Task,
			canWrite: true,
		},
		global: {
			plugins: [createPinia(), router, [VueQueryPlugin, {queryClient: new QueryClient()}]],
			mocks: {$t: (key: string) => key},
			stubs: {
				Icon: true,
				Modal: true,
			},
		},
	})
}

async function clickItem(wrapper: ReturnType<typeof mountMenu>, label: string) {
	await wrapper.get('.task-quick-actions__trigger').trigger('click')
	const item = wrapper.findAll('.dropdown-item').find(button => button.text() === label)
	expect(item, label).toBeDefined()
	await item!.trigger('click')
	await flushPromises()
}

describe('TaskQuickActions', () => {
	beforeEach(() => vi.clearAllMocks())

	it('adds only the current user, so other assignees stay', async () => {
		await clickItem(mountMenu([{id: 9}]), 'task.quickActions.assignMe')
		expect(sdk.taskAssigneesCreate).toHaveBeenCalledWith(expect.objectContaining({
			path: {task: 7},
			body: {user_id: 5},
		}))
	})

	it('removes only the current user when already assigned', async () => {
		await clickItem(mountMenu([{id: 5}, {id: 9}]), 'task.quickActions.unassignMe')
		expect(sdk.taskAssigneesDelete).toHaveBeenCalledWith(expect.objectContaining({
			path: {task: 7, user: 5},
		}))
		expect(sdk.taskAssigneesCreate).not.toHaveBeenCalled()
	})
})
