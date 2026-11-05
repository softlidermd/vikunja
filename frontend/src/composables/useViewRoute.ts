import {computed, reactive, shallowRef, watch} from 'vue'
import {useRoute, type RouteLocationNormalizedLoaded} from 'vue-router'

export type RouteQueryValue = string | string[] | null | undefined

// Behind the task modal the URL is the task's, so a view keeps reading its own query until the modal closes.
export function useViewRoute(): RouteLocationNormalizedLoaded {
	const route = useRoute()
	const viewQuery = shallowRef(route.query)
	watch(() => route.query, query => {
		if (route.name !== 'task.detail') viewQuery.value = query
	}, {flush: 'sync'})
	return reactive({
		params: computed(() => route.params),
		hash: computed(() => route.hash),
		query: computed(() => route.name === 'task.detail' ? viewQuery.value : route.query),
	}) as unknown as RouteLocationNormalizedLoaded
}
