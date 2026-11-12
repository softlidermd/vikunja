<template>
	<Multiselect
		ref="multiselect"
		v-model="labels"
		:loading="loading"
		:placeholder="$t('task.label.placeholder')"
		:multiple="true"
		:search-results="foundLabels"
		label="title"
		:creatable="creatable"
		:creation-disabled-message="creationDisabledMessage"
		:create-placeholder="$t('task.label.createPlaceholder')"
		:search-delay="10"
		show-empty
		:close-after-select="false"
		:disabled="disabled"
		:inline="inline"
		@search="findLabel"
		@select="addLabel"
		@create="createAndAddLabel"
	>
		<template #tag="{item: label}">
			<span
				:style="getLabelStyles(label)"
				class="tag"
			>
				<span>{{ label.title }}</span>
				<BaseButton
					v-if="!disabled"
					v-cy="'taskDetail.removeLabel'"
					:aria-label="$t('task.label.removeLabel', {label: label.title})"
					class="delete is-small"
					@click="removeLabel(label)"
				/>
			</span>
		</template>
		<template #searchResult="{option}">
			<span
				:style="getLabelStyles(option)"
				class="tag search-result"
			>
				<span>{{ option.title }}</span>
			</span>
		</template>
		<template #createOption="{query: newLabelTitle}">
			<span class="tag search-result">
				<span>{{ newLabelTitle }}</span>
			</span>
		</template>
	</Multiselect>
</template>

<script setup lang="ts">
import {ref, computed, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import {success} from '@/message'

import BaseButton from '@/components/base/BaseButton.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import type {Label} from '@/client/generated'
import {
	filterLabelsByQuery,
	getLabelByExactTitle,
	labelsInProject,
	refreshLabels,
	useCreateLabelMutation,
} from '@/client/queries/labels'
import {useAddTaskLabelMutation, useRemoveTaskLabelMutation} from '@/client/queries/taskMutations'
import {getRandomColorHex} from '@/helpers/color/randomColor'
import {useLabelStyles} from '@/composables/useLabelStyles'
import {useLabels} from '@/composables/useLabels'

const props = withDefaults(defineProps<{
	modelValue: Label[] | undefined
	taskId?: number
	projectId?: number
	disabled?: boolean
	creatable?: boolean
	creationDisabledMessage?: string
	inline?: boolean
}>(), {
	taskId: 0,
	projectId: 0,
	disabled: false,
	creatable: true,
	creationDisabledMessage: '',
	inline: false,
})

const emit = defineEmits<{
	'update:modelValue': [labels: Label[]],
}>()

const {t} = useI18n({useScope: 'global'})

const multiselect = ref<{focus: () => void} | null>(null)
const labels = ref<Label[]>([])
const query = ref('')

watch(
	() => props.modelValue,
	(value) => {
		labels.value = Array.from(new Map((value ?? []).map(label => [label.id, label])).values())
	},
	{
		immediate: true,
		deep: true,
	},
)

const addLabelMutation = useAddTaskLabelMutation()
const removeLabelMutation = useRemoveTaskLabelMutation()
const {labels: allLabels, isPending} = useLabels()
const createLabelMutation = useCreateLabelMutation()
const {getLabelStyles} = useLabelStyles()

// Only the task's project labels can be attached to it.
const projectLabels = computed(() => labelsInProject(allLabels.value, props.projectId))
// An empty query lists every label not yet on the task, so the picker shows options as soon as it is focused.
const foundLabels = computed(() => query.value === ''
	? projectLabels.value.filter(label => !labels.value.some(l => l.id === label.id))
	: filterLabelsByQuery(projectLabels.value, labels.value, query.value))
const loading = computed(() => isPending.value
	|| createLabelMutation.isPending.value
	|| addLabelMutation.isPending.value
	|| removeLabelMutation.isPending.value)

function findLabel(newQuery: string) {
	query.value = newQuery
}

async function addLabel(label: Label, showNotification = true) {
	if (props.taskId === 0) {
		emit('update:modelValue', labels.value)
		return
	}

	await addLabelMutation.mutateAsync({label: {...label, id: label.id!}, taskId: props.taskId})
	labels.value = Array.from(new Map(labels.value.map(label => [label.id, label])).values())
	if (showNotification) {
		success({message: t('task.label.addSuccess')})
	}
}

async function removeLabel(label: Label) {
	if (props.taskId !== 0) {
		await removeLabelMutation.mutateAsync({label: {...label, id: label.id!}, taskId: props.taskId})
	}

	const idx = labels.value.findIndex(l => l.id === label.id)
	if (idx !== -1) {
		labels.value.splice(idx, 1)
	}
	emit('update:modelValue', labels.value)
	success({message: t('task.label.removeSuccess')})
}

async function createAndAddLabel(title: string) {
	if (props.taskId === 0) {
		return
	}

	// The cached list may be stale: another user or tab could have created this label meanwhile.
	const existing = getLabelByExactTitle(labelsInProject(await refreshLabels(), props.projectId), title)
	if (existing) {
		if (!labels.value.some(label => label.id === existing.id)) {
			await addLabel(existing)
		}
		return
	}

	const newLabel = await createLabelMutation.mutateAsync({
		title,
		hex_color: getRandomColorHex(),
		project_id: props.projectId,
	})
	await addLabel(newLabel, false)
	success({message: t('task.label.addCreateSuccess')})
}

defineExpose({focus: () => multiselect.value?.focus()})
</script>

<style lang="scss" scoped>
.tag {
	margin: .25rem !important;
}

.tag.search-result {
	margin: 0 !important;
}

:deep(.input-wrapper) {
	padding: .25rem !important;
}

:deep(input.input) {
	padding: 0 .5rem;
}
</style>
