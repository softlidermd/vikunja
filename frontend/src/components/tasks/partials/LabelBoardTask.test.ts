import {describe, it, expect, vi} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createPinia} from 'pinia'
import {createRouter, createMemoryHistory} from 'vue-router'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'

import type {Task} from '@/client/generated'
import {PRIORITIES} from '@/constants/priorities'

vi.mock('@/message', () => ({error: vi.fn(), success: vi.fn()}))
vi.mock('vue-i18n', async importOriginal => ({
	...(await importOriginal<typeof import('vue-i18n')>()),
	useI18n: () => ({t: (key: string) => key}),
}))

import LabelBoardTask from './LabelBoardTask.vue'

const baseTask = {
	id: 7,
	project_id: 1,
	title: 'Write the invoice',
	done: false,
	labels: [{id: 1, title: 'Billing', hex_color: 'e8e8e8'}, {id: 2, title: 'Urgent', hex_color: 'ff0000'}],
} as Task

function mountRow(task: Partial<Task> = {}) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{path: '/', component: {render: () => null}},
			{path: '/tasks/:id', name: 'task.detail', component: {render: () => null}},
		],
	})
	const wrapper = mount(LabelBoardTask, {
		props: {task: {...baseTask, ...task}, canMarkAsDone: true, canEditStatus: true, hiddenLabelId: 1},
		global: {
			plugins: [createPinia(), router, [VueQueryPlugin, {queryClient: new QueryClient()}]],
			directives: {tooltip: {}},
			mocks: {$t: (key: string) => key},
			stubs: {
				FancyCheckbox: true,
				AssigneeList: true,
				CommentCount: true,
				TaskStatusSelect: true,
				TaskGlanceTooltip: {template: '<span><slot /></span>'},
			},
		},
	})
	return {wrapper, router}
}

describe('LabelBoardTask', () => {
	it('marks only high priorities and leaves out the card\'s own label', () => {
		const {wrapper} = mountRow({priority: PRIORITIES.MEDIUM})
		expect(wrapper.find('.board-task__priority').exists()).toBe(false)
		expect(wrapper.findAll('.board-task__dot').map(dot => dot.attributes('aria-label'))).toEqual(['Urgent'])

		expect(mountRow({priority: PRIORITIES.HIGH}).wrapper.find('.board-task__priority').exists()).toBe(true)
	})

	it('opens the task when the row is clicked', async () => {
		const {wrapper, router} = mountRow()
		await wrapper.get('.board-task__meta').trigger('click')
		await flushPromises()
		expect(router.currentRoute.value.fullPath).toBe('/tasks/7')
	})
})
