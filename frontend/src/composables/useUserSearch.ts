import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {keepPreviousData, useQuery, type Query} from '@tanstack/vue-query'
import {projectUserSearchQuery, userSearchKeys, userSearchQuery} from '@/client/queries/userSearch'
import type {User} from '@/client/generated'

// An empty search lists the users the server shows before typing, so pickers open with options.
export function useUserSearch(search: MaybeRefOrGetter<string>) {
	const query = useQuery(computed(() => ({
		...userSearchQuery(toValue(search)),
		placeholderData: keepPreviousData,
	})))
	return {...query, users: computed(() => query.data.value ?? [])}
}

export function useProjectUserSearch(projectId: MaybeRefOrGetter<number>, search: MaybeRefOrGetter<string>, enabled: MaybeRefOrGetter<boolean>) {
	const isEnabled = computed(() => toValue(enabled) && toValue(projectId) > 0)
	const query = useQuery(computed(() => ({
		...projectUserSearchQuery(toValue(projectId), toValue(search)),
		enabled: isEnabled.value,
		// Not keepPreviousData: it would show the previous project's users.
		placeholderData: (previousData: User[] | undefined, previousQuery: Query<User[], Error, User[], ReturnType<typeof userSearchKeys.project>> | undefined) => {
			if (!previousQuery) {
				return undefined
			}
			const [, , previousProjectId] = previousQuery.queryKey
			return previousProjectId === toValue(projectId) ? previousData : undefined
		},
	})))
	return {...query, users: computed(() => isEnabled.value ? query.data.value ?? [] : [])}
}
