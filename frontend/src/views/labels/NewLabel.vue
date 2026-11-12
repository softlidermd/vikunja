<template>
	<CreateEdit
		v-model:loading="loadingModel"
		:title="$t('label.create.title')"
		:primary-disabled="label.title === '' || !selectedProject?.id"
		@create="newLabel()"
	>
		<FormField
			id="labelTitle"
			v-model="label.title"
			v-focus
			:label="$t('label.attributes.title')"
			:disabled="showLoading"
			:loading="loading"
			:placeholder="$t('label.attributes.titlePlaceholder')"
			type="text"
			:error="showError && label.title === '' ? $t('label.create.titleRequired') : null"
			@keyup.enter="newLabel()"
		/>
		<FormField
			:label="$t('task.attributes.project')"
			:error="showError && !selectedProject?.id ? $t('label.create.projectRequired') : null"
		>
			<ProjectSearch
				v-model="selectedProject"
				show-empty
				:filter="isWritable"
			/>
		</FormField>
		<FormField :label="$t('label.attributes.color')">
			<ColorPicker v-model="label.hex_color" />
		</FormField>
	</CreateEdit>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useRouter} from 'vue-router'

import CreateEdit from '@/components/misc/CreateEdit.vue'
import ColorPicker from '@/components/input/ColorPicker.vue'
import FormField from '@/components/input/FormField.vue'
import ProjectSearch from '@/components/tasks/partials/ProjectSearch.vue'

import {useTitle} from '@/composables/useTitle'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {success} from '@/message'
import {getRandomColorHex} from '@/helpers/color/randomColor'
import {createLabelDraft, useCreateLabelMutation} from '@/client/queries/labels'
import type {ProjectResponse} from '@/client/queries/projects'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {PERMISSIONS} from '@/constants/permissions'

const router = useRouter()

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('label.create.title'))

const label = ref(createLabelDraft({hex_color: getRandomColorHex()}))
const createLabelMutation = useCreateLabelMutation()
const projectList = useProjects()
const authStore = useAuthStore()

const isWritable = (project: ProjectResponse) =>
	(project.max_permission ?? PERMISSIONS.READ) >= PERMISSIONS.READ_WRITE && !project.is_archived
const selectedProject = ref<ProjectResponse | null>(null)
// Projects may load after the page: preselect the default project once it is known, unless the user picked one.
watch(
	() => projectList.projects[authStore.settings.default_project_id ?? 0],
	defaultProject => {
		if (selectedProject.value === null && defaultProject && isWritable(defaultProject)) {
			selectedProject.value = defaultProject
		}
	},
	{immediate: true},
)

const showError = ref(false)
const loading = computed(() => createLabelMutation.isPending.value)
const showLoading = useDelayedLoading(loading)
const isSubmitting = ref(false)

const loadingModel = computed({
	get: () => isSubmitting.value || loading.value,
	set(value: boolean) {
		isSubmitting.value = value
	},
})

async function newLabel() {
	const projectId = selectedProject.value?.id
	if (label.value.title === '' || !projectId) {
		showError.value = true
		return
	}
	showError.value = false

	if (isSubmitting.value) {
		return
	}

	isSubmitting.value = true

	try {
		const newLabel = await createLabelMutation.mutateAsync({
			...label.value,
			project_id: projectId,
		})
		router.push({
			name: 'labels.index',
			params: {id: newLabel.id},
		})
		success({message: t('label.create.success')})
	} finally {
		isSubmitting.value = false
	}
}
</script>
