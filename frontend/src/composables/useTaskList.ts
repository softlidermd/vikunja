import {ref, shallowRef, watch, computed, type ComputedGetter} from 'vue'
import {useRouter, isNavigationFailure} from 'vue-router'
import type {LocationQueryRaw} from 'vue-router'
import {useRouteQuery} from '@vueuse/router'

import {
	getDefaultTaskFilterParams,
	type TaskExpansion,
	type TaskFilterParams,
} from '@/client/queries/tasks'
import {normalizePageNumber} from '@/client/queries/pagination'
import {isSavedFilterProject} from '@/client/queries/projects'
import {isRequestContextAbort} from '@/client/requestContext'
import {useTasks} from '@/composables/useTasks'
import {useClampedPage} from '@/composables/useClampedPage'
import {useViewRoute, type RouteQueryValue} from '@/composables/useViewRoute'
import {error} from '@/message'
import {useAuthStore} from '@/stores/auth'
import {useViewFiltersStore} from '@/stores/viewFilters'

export type Order = 'asc' | 'desc' | 'none'

export interface SortBy {
	id?: Order
	index?: Order
	done?: Order
	title?: Order
	priority?: Order
	size?: Order
	due_date?: Order
	start_date?: Order
	end_date?: Order
	percent_done?: Order
	created?: Order
	updated?: Order
	done_at?: Order,
	position?: Order,
}

const VALID_SORT_FIELDS = new Set<string>(
	['id', 'index', 'done', 'title', 'priority', 'size', 'due_date', 'start_date',
		'end_date', 'percent_done', 'created', 'updated', 'done_at', 'position'],
)

function parseSortQuery(raw: string, fallback: SortBy): SortBy {
	const result: Record<string, Order> = {}
	for (const part of raw.split(',')) {
		const [field, order] = part.split(':')
		if (!VALID_SORT_FIELDS.has(field)) continue
		if (order !== 'asc' && order !== 'desc') continue
		result[field] = order
	}
	return Object.keys(result).length > 0 ? result as SortBy : {...fallback}
}

function serializeSortBy(sortBy: SortBy, defaultSort: SortBy): string | undefined {
	const keys = Object.keys(sortBy) as (keyof SortBy)[]
	const defaultKeys = Object.keys(defaultSort) as (keyof SortBy)[]
	const isDefault = keys.length === defaultKeys.length &&
		keys.every(k => sortBy[k] === defaultSort[k])
	if (isDefault) return undefined
	return keys.map(k => `${k}:${sortBy[k]}`).join(',')
}

const SORT_BY_DEFAULT: SortBy = {
	id: 'desc',
}

interface TaskListQueryState {
	sort: string | undefined
	filter: string | undefined
	s: string | undefined
	page: number
}

export function buildStoredQuery(state: TaskListQueryState): LocationQueryRaw {
	const query: LocationQueryRaw = {}
	if (state.sort) query.sort = state.sort
	if (state.filter) query.filter = state.filter
	if (state.s) query.s = state.s
	if (state.page > 1) query.page = String(state.page)
	return query
}

// The id is only a tie-breaker for the explicitly selected sort fields.
export function formatSortOrder(sortBy: SortBy, params: TaskFilterParams): TaskFilterParams {
	const sortKeys = Object.keys(sortBy) as (keyof SortBy)[]
	const idIndex = sortKeys.indexOf('id')
	if (idIndex >= 0) {
		sortKeys.splice(idIndex, 1)
		sortKeys.push('id')
	}
	params.sort_by = sortKeys
	params.order_by = sortKeys.map(key => sortBy[key] ?? 'none')
	return params
}

/**
 * A project view's sort, filter, search and page in the URL, remembered per view so they
 * come back when the view is opened again without a query (tab links, closing a task).
 */
export function useViewQuery(
	projectViewIdGetter: ComputedGetter<number>,
	sortByDefault: SortBy,
	defaultFilterGetter: () => string | undefined = () => undefined,
) {
	const projectViewId = computed(() => projectViewIdGetter())
	const router = useRouter()
	const viewFiltersStore = useViewFiltersStore()

	const route = useViewRoute()
	const page = useRouteQuery('page', '1', { transform: normalizePageNumber, route })
	const filter = useRouteQuery<RouteQueryValue>('filter', undefined, {route})
	const s = useRouteQuery<RouteQueryValue>('s', undefined, {route})

	const sortQuery = useRouteQuery<RouteQueryValue>('sort', undefined, {route})

	const sortBy = computed<SortBy>({
		get() {
			const raw = sortQuery.value as string | undefined
			if (!raw) return {...sortByDefault}
			return parseSortQuery(raw, sortByDefault)
		},
		set(val: SortBy) {
			sortQuery.value = serializeSortBy(val, sortByDefault) || undefined
		},
	})

	const pendingQueryRestore = shallowRef<Promise<unknown>>()
	// Sidebar links omit the query, and project views are reused across navigation.
	const syncedViewId = shallowRef<number>()
	watch(
		[projectViewId, sortQuery, filter, s, page],
		([viewId, sortValue, filterValue, sValue, pageValue]) => {
			const viewIdChanged = viewId !== syncedViewId.value
			syncedViewId.value = viewId

			const defaultFilter = defaultFilterGetter()
			// A filter the user cleared stays in the URL as `filter=` (''), so only navigation empties it.
			const urlIsEmpty = !sortValue && filterValue == null && !sValue && pageValue === 1
			if (urlIsEmpty && (viewIdChanged || defaultFilter)) {
				// Reloaded behind the task modal: the URL is the task's, so neither restore into it nor store it.
				if (router.currentRoute.value.name === 'task.detail') return
				const stored = viewFiltersStore.getViewQuery(viewId)
				// The default fills in until a filter (even a cleared one) is stored.
				const storedQuery = defaultFilter && !('filter' in stored) ? {filter: defaultFilter, ...stored} : stored
				if (Object.keys(storedQuery).length > 0) {
					const restore = router.replace({query: {...router.currentRoute.value.query, ...storedQuery}})
					pendingQueryRestore.value = restore
					restore
						.catch(failure => {
							if (!isNavigationFailure(failure)) throw failure
						})
						.finally(() => {
							if (pendingQueryRestore.value === restore) {
								pendingQueryRestore.value = undefined
							}
						})
					return
				}
			}

			const query = buildStoredQuery({
				sort: sortValue as string | undefined,
				filter: filterValue as string | undefined,
				s: sValue as string | undefined,
				page: pageValue,
			})
			if (filterValue === '') query.filter = ''
			if (Object.keys(query).length > 0) {
				viewFiltersStore.setViewQuery(viewId, query)
			} else {
				viewFiltersStore.clearViewQuery(viewId)
			}
		},
		{immediate: true},
	)

	// Hold fetches while a stored query is being restored, or before this view's query was read.
	const isReady = computed(() => !pendingQueryRestore.value && projectViewId.value === syncedViewId.value)

	return {page, filter, s, sortQuery, sortBy, isReady}
}

/**
 * This mixin provides a base set of methods and properties to get tasks.
 */
export function useTaskList(
	projectIdGetter: ComputedGetter<number>,
	projectViewIdGetter: ComputedGetter<number>,
	sortByDefault: SortBy = SORT_BY_DEFAULT,
	expandGetter: ComputedGetter<TaskExpansion> = () => ['subtasks'],
) {
	
	const projectId = computed(() => projectIdGetter())
	const projectViewId = computed(() => projectViewIdGetter())

	// Saved filters hide the filter popup and AND the request filter into their own, so no default there.
	const defaultFilter = () => isSavedFilterProject({id: projectId.value}) ? undefined : 'done = false'
	const {page, filter, s, sortQuery, sortBy, isReady} = useViewQuery(projectViewIdGetter, sortByDefault, defaultFilter)

	const params = ref<TaskFilterParams>({...getDefaultTaskFilterParams()})

	// sync: a restored query must reach params in the same flush as its page, or the page reset below fires.
	watch(filter, v => { params.value.filter = String(v ?? '') }, { immediate: true, flush: 'sync' })
	watch(s, v => { params.value.q = String(v ?? '') }, { immediate: true, flush: 'sync' })

	watch(() => params.value.filter, v => {
		// Skip echoes of the URL itself, so a view switch doesn't write `filter=` into the next view.
		if ((v || undefined) === (filter.value || undefined)) return
		filter.value = v || (defaultFilter() ? '' : undefined)
	})
	watch(() => params.value.q, v => { s.value = v || undefined })

	const allParams = computed(() => {
		const loadParams = {...params.value}

		// Relevance ranking only engages when no sort is sent, so omit the default
		// sort while searching and let an explicit user sort still take precedence.
		if (loadParams.q && !sortQuery.value) {
			loadParams.sort_by = []
			loadParams.order_by = []
			return loadParams
		}

		return formatSortOrder(sortBy.value, loadParams)
	})

	watch(
		[params, sortBy, page],
		([, , newPage], [, , oldPage]) => {
			// A redundant page write can cancel the navigation restoring a saved sort.
			if (newPage === oldPage && newPage !== 1) {
				page.value = 1
			}
		},
		{deep: true},
	)
	
	const authStore = useAuthStore()
	
	const scope = computed(() => ({
		project: projectId.value,
		view: projectViewId.value,
		params: {
			...allParams.value,
			filter_timezone: authStore.settings.timezone,
			expand: expandGetter(),
		},
	}))
	const query = useTasks(scope, {
		page,
		// Scope updates before the restore decision is made; hold the fetch until then.
		enabled: () => isReady.value,
	})
	const loading = query.isFetching
	const {tasks, totalPages} = query

	// A remembered or linked page can lie past the end (fewer tasks, a larger page size); it would render
	// empty without pagination to leave it.
	useClampedPage(page, query)

	watch(query.error, cause => {
		if (cause && !isRequestContextAbort(cause)) error(cause)
	})

	async function loadTasks() {
		await query.refetch()
		return tasks.value
	}

	return {
		tasks,
		loading,
		totalPages,
		currentPage: page,
		loadTasks,
		params,
		sortByParam: sortBy,
	}
}
