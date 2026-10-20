import {shallowMount, flushPromises} from '@vue/test-utils'
import {describe, expect, it, vi, beforeEach} from 'vitest'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'
import {createRouter, createMemoryHistory} from 'vue-router'
import {createPinia} from 'pinia'

import type {Task} from '@/client/generated'
import {removeToken} from '@/helpers/auth'
import PrioritySelect from '@/components/tasks/partials/PrioritySelect.vue'
import EditLabels from '@/components/tasks/partials/EditLabels.vue'
import BaseButton from '@/components/base/BaseButton.vue'

const sdk = vi.hoisted(() => ({
	patchTasksRead: vi.fn(),
	projectViewTasksList: vi.fn(),
	projectsList: vi.fn(),
}))

vi.mock('@/client/generated', () => sdk)
vi.mock('@/stores/auth', () => ({
	useAuthStore: () => ({
		settings: {timezone: 'UTC'},
		isLinkShareAuth: false,
	}),
}))
vi.mock('@/message', () => ({
	error: vi.fn(),
	success: vi.fn(),
}))
vi.mock('@/composables/useCurrentProject', () => ({
	useCurrentProject: () => ({
		currentProject: {value: {
			id: 1,
			max_permission: 2,
		}},
		isPending: {value: false},
	}),
}))
vi.mock('vue-i18n', async importOriginal => ({
	...await importOriginal<typeof import('vue-i18n')>(),
	useI18n: () => ({t: (key: string) => key}),
}))

import ProjectTable from './ProjectTable.vue'

// What the server has stored; the task list refetch after a save answers with it.
let serverTask: Task

function deferPatch() {
	let resolve!: () => void
	sdk.patchTasksRead.mockImplementationOnce(({body}: {body: {path: string, value: unknown}[]}) => new Promise(r => {
		resolve = () => {
			for (const {path, value} of body) {
				serverTask = {
					...serverTask,
					[path.slice(1)]: value,
				}
			}
			r({data: serverTask})
		}
	}))
	return () => resolve()
}

async function mountTable() {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [{
			path: '/',
			component: {render: () => null},
		}],
	})
	await router.push('/')
	const wrapper = shallowMount(ProjectTable, {
		props: {
			isLoadingProject: false,
			projectId: 1,
			viewId: 10,
		},
		global: {
			plugins: [
				router,
				createPinia(),
				[VueQueryPlugin, {queryClient: new QueryClient({defaultOptions: {queries: {retry: false}}})}],
			],
			mocks: {$t: (key: string) => key},
			stubs: {
				ProjectWrapper: {template: '<div><slot name="default"/></div>'},
				Popup: {template: '<div><slot name="content" :is-open="true"/></div>'},
			},
		},
	})
	await flushPromises()
	return wrapper
}

describe('ProjectTable inline saves', () => {
	beforeEach(() => {
		localStorage.setItem('tableViewColumns', JSON.stringify({
			title: true,
			priority: true,
		}))
		serverTask = {
			id: 1,
			title: 'Task 1',
			project_id: 1,
			priority: 0,
			assignees: [],
			labels: [],
		} as Task
		sdk.patchTasksRead.mockReset()
		sdk.projectsList.mockResolvedValue({data: {
			items: [],
			page: 1,
			total_pages: 1,
		}})
		sdk.projectViewTasksList.mockImplementation(async () => ({data: {
			items: [serverTask],
			page: 1,
			total_pages: 1,
		}}))
	})

	it('sends the second save of a task only after the first one resolved', async () => {
		const wrapper = await mountTable()
		const resolveFirst = deferPatch()
		const resolveSecond = deferPatch()

		wrapper.findComponent(PrioritySelect).vm.$emit('update:modelValue', 1)
		wrapper.findComponent(PrioritySelect).vm.$emit('update:modelValue', 3)
		await flushPromises()
		expect(sdk.patchTasksRead).toHaveBeenCalledTimes(1)

		resolveFirst()
		await flushPromises()
		expect(sdk.patchTasksRead).toHaveBeenCalledTimes(2)

		resolveSecond()
		await flushPromises()
		expect(wrapper.findComponent(PrioritySelect).props('modelValue')).toBe(3)
	})

	it('keeps the label picker of the first task on that task after another row opens it', async () => {
		// Saved columns are merged over the defaults, so the assignee buttons must be switched off explicitly.
		localStorage.setItem('tableViewColumns', JSON.stringify({
			labels: true,
			assignees: false,
		}))
		sdk.projectViewTasksList.mockImplementation(async () => ({data: {
			items: [
				serverTask,
				{
					...serverTask,
					id: 2,
					title: 'Task 2',
				},
			],
			page: 1,
			total_pages: 1,
		}}))
		const wrapper = await mountTable()
		const [firstRow, secondRow] = wrapper.findAllComponents(BaseButton)

		firstRow!.vm.$emit('click', {currentTarget: firstRow!.element})
		await flushPromises()
		const firstPicker = wrapper.findComponent(EditLabels)
		secondRow!.vm.$emit('click', {currentTarget: secondRow!.element})
		await flushPromises()

		// A label still being created in the first picker must not be added to task 2.
		expect(firstPicker.props('taskId')).toBe(1)
		expect(wrapper.findComponent(EditLabels).props('taskId')).toBe(2)
	})

	it('drops a queued save once the user logged out', async () => {
		const wrapper = await mountTable()
		const resolveFirst = deferPatch()

		wrapper.findComponent(PrioritySelect).vm.$emit('update:modelValue', 1)
		wrapper.findComponent(PrioritySelect).vm.$emit('update:modelValue', 3)
		await flushPromises()
		removeToken()
		resolveFirst()
		await flushPromises()

		expect(sdk.patchTasksRead).toHaveBeenCalledTimes(1)
	})
})
