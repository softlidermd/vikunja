<template>
	<Multiselect
		class="control is-expanded"
		:placeholder="$t('project.search')"
		:search-results="foundProjects"
		:show-empty="showEmpty"
		label="title"
		:select-placeholder="$t('project.searchSelect')"
		:model-value="project"
		@update:modelValue="(val) => val === null ? select(null) : Object.assign(project, val)"
		@select="select"
		@search="findProjects"
	>
		<template #searchResult="{option}">
			<span
				v-if="projectList.getAncestors(option).length > 1"
				class="has-text-grey"
			>
				{{ projectList.getAncestors(option).slice(0, -1).map(p => getProjectTitle(p)).join(' &gt; ') }} &gt;
			</span>
			{{ getProjectTitle(option) }}
		</template>
	</Multiselect>
</template>

<script lang="ts" setup>
import {reactive, ref, watch} from 'vue'

import {normalizeProject, type ProjectResponse} from '@/client/queries/projects'
import {useProjects} from '@/composables/useProjects'
import {getProjectTitle} from '@/helpers/getProjectTitle'

import Multiselect from '@/components/input/Multiselect.vue'

const props = withDefaults(defineProps<{
	modelValue?: ProjectResponse | null
	savedFiltersOnly?: boolean
	/** List the real, non-archived projects while the query is empty. */
	showEmpty?: boolean
	filter?: (project: ProjectResponse) => boolean,
}>(), {
	modelValue: () => normalizeProject({id: 0}),
	savedFiltersOnly: false,
	showEmpty: false,
	filter: () => true,
})

const emit = defineEmits<{
	'update:modelValue': [value: ProjectResponse | null]
}>()

const project = reactive<ProjectResponse>(normalizeProject({id: 0}))

watch(
	() => props.modelValue,
	(newProject) => Object.assign(project, newProject ?? normalizeProject({id: 0})),
	{
		immediate: true,
		deep: true,
	},
)

const projectList = useProjects()

const listAll = () => projectList.projectsArray.filter(p => p.id > 0 && !p.is_archived && props.filter(p))

const foundProjects = ref<ProjectResponse[]>([])
if (props.showEmpty) {
	foundProjects.value = listAll()
}
function findProjects(query: string) {
	if (query === '') {
		select(null)
		if (props.showEmpty) {
			foundProjects.value = listAll()
			return
		}
	}
	
	if (props.savedFiltersOnly) {
		const found = projectList.searchSavedFilter(query)
		foundProjects.value = found.filter(props.filter)
		return
	}
	
	const found = projectList.searchProject(query)
	foundProjects.value = found.filter(props.filter)
}

function select(p: ProjectResponse | null) {
	if (p === null) {
		Object.assign(project, normalizeProject({id: 0}))
		emit('update:modelValue', null)
		return
	}
	Object.assign(project, p)
	emit('update:modelValue', project)
}
</script>
