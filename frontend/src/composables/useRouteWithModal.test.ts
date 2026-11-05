import {describe, it, expect, vi} from 'vitest'
import {defineComponent} from 'vue'
import {mount} from '@vue/test-utils'
import {createRouter, createMemoryHistory} from 'vue-router'
import {QueryClient, VueQueryPlugin} from '@tanstack/vue-query'

vi.mock('@/stores/base', () => ({useBaseStore: () => ({currentProjectId: 0})}))
vi.mock('@/composables/useProjects', () => ({useProjects: () => ({projects: {}})}))

import {useRouteWithModal} from './useRouteWithModal'
import {useViewRoute} from './useViewRoute'

const empty = {render: () => null}

async function setup(start: string) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{
				path: '/',
				name: 'home',
				component: empty,
			},
			{
				path: '/projects/:projectId/:viewId',
				name: 'project.view',
				component: empty,
			},
			{
				path: '/tasks/:id',
				name: 'task.detail',
				component: empty,
				props: route => ({taskId: Number(route.params.id)}),
			},
		],
	})
	await router.push(start)
	await router.isReady()
	let modal!: ReturnType<typeof useRouteWithModal>
	let viewRoute!: ReturnType<typeof useViewRoute>
	mount(defineComponent({
		setup() {
			modal = useRouteWithModal()
			viewRoute = useViewRoute()
			return () => null
		},
	}), {
		global: {
			plugins: [
				router,
				[VueQueryPlugin, {queryClient: new QueryClient()}],
			],
		},
	})
	return {router, modal, viewRoute}
}

describe('useRouteWithModal', () => {
	it('opens a task linked without a backdrop over the page it was opened from', async () => {
		const {router, modal} = await setup('/projects/1/2')
		await router.push('/tasks/7')
		expect(modal.routeWithModal.value.fullPath).toBe('/projects/1/2')
		expect(modal.currentModal.value).toBeDefined()
	})

	it('opens a directly loaded task over home', async () => {
		const {modal} = await setup('/tasks/7')
		expect(modal.routeWithModal.value.name).toBe('home')
		expect(modal.currentModal.value).toBeDefined()
	})

	it('keeps the page as backdrop when another task opens from inside the modal', async () => {
		const {router, modal} = await setup('/projects/1/2')
		await router.push('/tasks/7')
		await router.push({
			path: '/tasks/8',
			state: {backdropView: '/tasks/7'},
		})
		expect(modal.routeWithModal.value.fullPath).toBe('/projects/1/2')
	})

	it('keeps the view query behind the task modal', async () => {
		const {router, viewRoute} = await setup('/projects/1/2?page=3&filter=done%3Dfalse')
		await router.push('/tasks/7')
		expect(viewRoute.query).toEqual({
			page: '3',
			filter: 'done=false',
		})
		await router.push('/projects/1/2?page=4')
		expect(viewRoute.query).toEqual({page: '4'})
	})
})
