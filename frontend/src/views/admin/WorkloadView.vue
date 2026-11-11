<template>
	<Card :title="$t('admin.workload.title')">
		<p v-if="isPending">
			{{ $t('misc.loading') }}
		</p>
		<template v-else-if="data">
			<section
				class="workload__block"
				data-testid="workload-total"
			>
				<h2 class="workload__name">
					{{ $t('admin.workload.companyTotal') }}
				</h2>
				<ul class="workload__stats">
					<li
						v-for="stat in stats(data.total)"
						:key="stat"
					>
						{{ stat }}
					</li>
				</ul>
			</section>
			<section
				v-for="block in blocks"
				:key="block.key"
				class="workload__block"
				data-testid="workload-block"
			>
				<h2 class="workload__name">
					{{ block.person.user ? getDisplayName(block.person.user) : $t('admin.workload.unassigned') }}
				</h2>
				<ul class="workload__stats">
					<li
						v-for="stat in stats(block.person)"
						:key="stat"
					>
						{{ stat }}
					</li>
				</ul>
				<div
					v-if="block.person.tasks?.length"
					class="has-horizontal-overflow"
				>
					<table class="table is-striped is-hoverable is-fullwidth">
						<thead>
							<tr>
								<th>{{ $t('task.attributes.project') }}</th>
								<th>{{ $t('task.attributes.title') }}</th>
								<th>{{ $t('task.attributes.status') }}</th>
								<th>{{ $t('task.attributes.priority') }}</th>
								<th>{{ $t('task.attributes.size') }}</th>
								<th>{{ $t('task.attributes.dueDate') }}</th>
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="task in block.person.tasks"
								:key="task.id"
							>
								<td>{{ task.project_title }}</td>
								<td>
									<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">
										{{ task.title }}
									</RouterLink>
								</td>
								<td>{{ task.status }}</td>
								<td>
									<PriorityLabel
										:priority="task.priority ?? 0"
										:show-all="true"
									/>
								</td>
								<td>{{ $t(taskSizeLabelKey(task.size ?? 0)) }}</td>
								<td>
									<TimeDisplay :date="parseDateOrNull(task.due_date)" />
								</td>
							</tr>
						</tbody>
					</table>
				</div>
				<p v-else>
					{{ $t('admin.workload.empty') }}
				</p>
			</section>
		</template>
	</Card>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {useQuery} from '@tanstack/vue-query'

import type {WorkloadTotals} from '@/client/generated'
import {adminWorkloadQuery} from '@/client/queries/admin'
import Card from '@/components/misc/Card.vue'
import TimeDisplay from '@/components/misc/TimeDisplay.vue'
import PriorityLabel from '@/components/tasks/partials/PriorityLabel.vue'
import {taskSizeLabelKey} from '@/constants/taskSizes'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {getDisplayName} from '@/helpers/user'

const {t} = useI18n({useScope: 'global'})

const {data, isPending} = useQuery(adminWorkloadQuery())

// The unassigned block comes last; its user is null at runtime, whatever the generated type says.
const blocks = computed(() => [
	...(data.value?.people ?? []).map(person => ({key: `user-${person.user?.id}`, person})),
	...(data.value?.unassigned ? [{key: 'unassigned', person: data.value.unassigned}] : []),
])

function stats(totals: WorkloadTotals | undefined) {
	return [
		t('admin.workload.hours', {hours: totals?.hours ?? 0}),
		`${t('task.size.unset')}: ${totals?.unestimated ?? 0}`,
		`${t('task.size.long')}: ${totals?.long ?? 0}`,
		`${t('admin.workload.ideas')}: ${totals?.ideas ?? 0}`,
	]
}
</script>

<style lang="scss" scoped>
.workload__block + .workload__block {
	margin-block-start: 2rem;
}

.workload__name {
	font-size: 1.25rem;
	font-weight: 600;
	margin-block-end: 0.25rem;
}

.workload__stats {
	display: flex;
	flex-wrap: wrap;
	gap: 0.25rem 1rem;
	margin-block-end: 0.75rem;
	color: var(--grey-600);
}
</style>
