import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'

import {
	filterLabelsByQuery as filterLabels,
	getLabelById as findLabelById,
	getLabelIdsByTitle as findLabelIdsByTitle,
	getLabelsByIds as findLabelsByIds,
	labelsQuery,
} from '@/client/queries/labels'
import type {Label} from '@/client/generated'

export function useLabels() {
	const query = useQuery(labelsQuery())
	const labels = computed(() => query.data.value ?? [])

	return {
		labels,
		isPending: query.isPending,
		filterLabelsByQuery: (hidden: Label[], value: string) => filterLabels(labels.value, hidden, value),
		getLabelById: (id: number) => findLabelById(labels.value, id),
		getLabelsByIds: (ids: number[]) => findLabelsByIds(labels.value, ids),
		getLabelIdsByTitle: (title: string) => findLabelIdsByTitle(labels.value, title),
	}
}
