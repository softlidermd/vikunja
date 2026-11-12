import {shallowMount} from '@vue/test-utils'
import {defineComponent, nextTick} from 'vue'
import {beforeEach, describe, expect, it, vi} from 'vitest'

const labelState = vi.hoisted(() => ({
	labels: undefined as {value: Array<{id?: number, title?: string, project_id?: number}>} | undefined,
	isPending: undefined as {value: boolean} | undefined,
}))

const projectState = vi.hoisted(() => ({
	projects: {} as Record<number, {id: number, title: string}>,
}))

vi.mock('@/composables/useLabels', async () => {
	const {ref} = await import('vue')
	const {getLabelIdsByTitle} = await import('@/client/queries/labels')
	labelState.labels = ref([])
	labelState.isPending = ref(true)

	return {
		useLabels: () => ({
			labels: labelState.labels,
			isPending: labelState.isPending,
			getLabelById: (id: number) => labelState.labels?.value.find(label => label.id === id),
			getLabelIdsByTitle: (title: string) => getLabelIdsByTitle(labelState.labels?.value ?? [], title),
		}),
	}
})

vi.mock('@/composables/useProjects', () => ({
	useProjects: () => ({
		projects: projectState.projects,
		findProjectByExactname: (title: string) =>
			Object.values(projectState.projects).find(project => project.title === title) ?? null,
	}),
}))

vi.mock('vue-i18n', async importOriginal => ({
	...await importOriginal<typeof import('vue-i18n')>(),
	useI18n: () => ({t: (key: string) => key}),
}))

const FilterInputStub = defineComponent({
	name: 'FilterInput',
	props: {modelValue: {type: String, default: ''}},
	emits: ['update:modelValue'],
	template: '<div />',
})

import ViewEditForm from './ViewEditForm.vue'
import XButton from '@/components/input/Button.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import type {ProjectView} from '@/client/generated'

function mountForm(modelValue: ProjectView = {
	id: 1,
	project_id: 1,
	title: 'List',
	view_kind: 'kanban',
	bucket_configuration_mode: 'filter',
	filter: {filter: 'labels = 1', filter_include_nulls: false},
	bucket_configuration: [{
		title: 'Bucket',
		filter: {filter: 'labels = 1', filter_include_nulls: false},
	}],
}) {
	return shallowMount(ViewEditForm, {
		props: {
			modelValue,
		},
		global: {
			stubs: {FilterInput: FilterInputStub},
			mocks: {$t: (key: string) => key},
			directives: {focus: () => {}},
		},
	})
}

describe('ViewEditForm', () => {
	beforeEach(() => {
		labelState.labels!.value = []
		labelState.isPending!.value = true
		for (const id of Object.keys(projectState.projects)) {
			delete projectState.projects[Number(id)]
		}
	})

	it('resolves saved label ids after labels finish loading', async () => {
		const wrapper = mountForm()
		expect(wrapper.findAllComponents(FilterInputStub).map(input => input.props('modelValue')))
			.toEqual(['labels = 1', 'labels = 1'])

		labelState.labels!.value = [{id: 1, title: 'Work'}]
		labelState.isPending!.value = false
		await nextTick()

		expect(wrapper.findAllComponents(FilterInputStub).map(input => input.props('modelValue')))
			.toEqual(['labels = Work', 'labels = Work'])
	})

	it('keeps a filter edited before labels finish loading', async () => {
		const wrapper = mountForm()
		wrapper.findAllComponents(FilterInputStub)[0].vm.$emit('update:modelValue', 'labels = Personal')
		await nextTick()

		labelState.labels!.value = [{id: 1, title: 'Work'}]
		labelState.isPending!.value = false
		await nextTick()

		expect(wrapper.findAllComponents(FilterInputStub)[0].props('modelValue')).toBe('labels = Personal')
	})

	it('preserves model sorting when saving', async () => {
		const modelValue: ProjectView = {
			id: 1,
			project_id: 1,
			title: 'List',
			view_kind: 'kanban',
			bucket_configuration_mode: 'filter',
			filter: {
				sort_by: ['done', 'id'],
				order_by: ['asc', 'desc'],
				filter: 'done = false',
				filter_include_nulls: true,
				s: '',
			},
			bucket_configuration: [{
				title: 'Bucket',
				filter: {
					sort_by: ['position'],
					order_by: ['desc'],
					filter: 'done = false',
					filter_include_nulls: false,
					s: '',
				},
			}],
		}
		const wrapper = mountForm(modelValue)

		await wrapper.find('form').trigger('submit')

		const emitted = wrapper.emitted('update:modelValue')
		const saved = emitted?.[emitted.length - 1]?.[0] as ProjectView
		expect(saved.filter).toMatchObject({
			sort_by: ['done', 'id'],
			order_by: ['asc', 'desc'],
		})
		expect(saved.bucket_configuration?.[0].filter).toMatchObject({
			sort_by: ['position'],
			order_by: ['desc'],
		})
	})

	it('does not save while focus moves to another control in the form', async () => {
		const wrapper = mountForm({
			id: 1,
			project_id: 1,
			title: 'Board',
			view_kind: 'kanban',
			bucket_configuration_mode: 'manual',
		})
		const manualMode = wrapper.find<HTMLInputElement>('input[value="manual"]')
		const filterMode = wrapper.find<HTMLInputElement>('input[value="filter"]')

		await manualMode.trigger('focusout', {relatedTarget: filterMode.element})

		expect(wrapper.emitted('update:modelValue')).toBeUndefined()
		await filterMode.trigger('click')
		expect(filterMode.element.checked).toBe(true)
		expect(wrapper.emitted('update:modelValue')).toBeUndefined()
	})

	it('saves when focus leaves the form without explicit save buttons', async () => {
		const wrapper = mountForm()

		await wrapper.find('form').trigger('focusout', {relatedTarget: document.body})

		expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
	})
})

// Label 14 is a migration clone in project 2; label 30 is an unrelated label with the same title in project 3.
describe('ViewEditForm stored label filters', () => {
	const bugLabels = [
		{id: 14, title: 'Bug', project_id: 2},
		{id: 30, title: 'Bug', project_id: 3},
	]

	function bugView(filter = 'labels in 14'): ProjectView {
		return {
			id: 1,
			project_id: 1,
			title: 'Board',
			view_kind: 'kanban',
			bucket_configuration_mode: 'filter',
			filter: {filter, filter_include_nulls: false},
			bucket_configuration: [
				{
					title: 'Clone',
					filter: {filter: 'labels = 14', filter_include_nulls: false},
				},
				{
					title: 'Unrelated',
					filter: {filter: 'labels = 30', filter_include_nulls: false},
				},
			],
		}
	}

	async function loadLabels(labels = bugLabels) {
		labelState.labels!.value = labels
		labelState.isPending!.value = false
		await nextTick()
	}

	async function save(wrapper: ReturnType<typeof mountForm>): Promise<ProjectView> {
		await wrapper.find('form').trigger('submit')
		const emitted = wrapper.emitted('update:modelValue')!
		return emitted[emitted.length - 1][0] as ProjectView
	}

	const filters = (view: ProjectView) => [
		view.filter?.filter,
		...(view.bucket_configuration ?? []).map(bucket => bucket.filter?.filter),
	]

	const inputs = (wrapper: ReturnType<typeof mountForm>) => wrapper.findAllComponents(FilterInputStub)

	beforeEach(() => {
		labelState.labels!.value = []
		labelState.isPending!.value = true
	})

	it('(a) saves every untouched filter unchanged', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()
		expect(inputs(wrapper).map(input => input.props('modelValue')))
			.toEqual(['labels in Bug', 'labels = Bug', 'labels = Bug'])

		const saved = await save(wrapper)

		expect(filters(saved)).toEqual(['labels in 14', 'labels = 14', 'labels = 30'])
		expect(saved.filter).not.toHaveProperty('apiFilter')
		expect(saved.filter).not.toHaveProperty('shownText')
		expect(saved.bucket_configuration?.[0].filter).not.toHaveProperty('apiFilter')
	})

	it('(b) resolves only the edited bucket to every label with the title', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		inputs(wrapper)[2].vm.$emit('update:modelValue', 'labels = Bug && done = false')
		const saved = await save(wrapper)

		expect(filters(saved)).toEqual(['labels in 14', 'labels = 14', 'labels in 14, 30 && done = false'])
	})

	it('(c) keeps an unresolved label id through an edit', async () => {
		const wrapper = mountForm(bugView('labels = 999'))
		await loadLabels()
		expect(inputs(wrapper)[0].props('modelValue')).toBe('labels = 999')

		inputs(wrapper)[0].vm.$emit('update:modelValue', 'labels = 999 && done = false')
		const saved = await save(wrapper)

		expect(saved.filter?.filter).toBe('labels = 999 && done = false')
	})

	it('(d) saves unchanged while labels are still loading', async () => {
		const wrapper = mountForm(bugView())

		const saved = await save(wrapper)

		expect(filters(saved)).toEqual(['labels in 14', 'labels = 14', 'labels = 30'])
	})

	it('(e) keeps the remaining bucket exact after deleting the one before it', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		await wrapper.find('button.is-danger').trigger('click')
		const saved = await save(wrapper)

		expect(saved.bucket_configuration?.map(bucket => bucket.title)).toEqual(['Unrelated'])
		expect(filters(saved)).toEqual(['labels in 14', 'labels = 30'])
	})

	it('(f) saves a pushed empty bucket with an empty filter', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		wrapper.findAllComponents(XButton).find(button => button.props('icon') === 'plus')!.vm.$emit('click')
		await nextTick()
		const saved = await save(wrapper)

		expect(saved.bucket_configuration).toHaveLength(3)
		expect(saved.bucket_configuration?.[2].filter).toEqual({
			sort_by: [],
			order_by: [],
			filter: '',
			filter_include_nulls: false,
			s: '',
		})
		expect(filters(saved).slice(0, 3)).toEqual(['labels in 14', 'labels = 14', 'labels = 30'])
	})

	it('(g) saves an include-nulls toggle without touching the filter string', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		wrapper.findAllComponents(FancyCheckbox)[0].vm.$emit('update:modelValue', true)
		await nextTick()
		const saved = await save(wrapper)

		expect(saved.filter).toMatchObject({filter: 'labels in 14', filter_include_nulls: true})
	})

	it('(h) keeps the ids of an untouched filter when a refetch renames a label', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		labelState.labels!.value = [
			{id: 14, title: 'Defect', project_id: 2},
			{id: 30, title: 'Bug', project_id: 3},
		]
		await nextTick()
		const saved = await save(wrapper)

		expect(filters(saved)).toEqual(['labels in 14', 'labels = 14', 'labels = 30'])
	})

	it('(i) saves an untouched project filter unchanged when projects load after labels', async () => {
		const wrapper = mountForm(bugView('project = 7 && labels = 14'))
		await loadLabels()
		expect(inputs(wrapper)[0].props('modelValue')).toBe('project = 7 && labels = Bug')

		projectState.projects[7] = {id: 7, title: 'Seven'}
		await nextTick()
		const saved = await save(wrapper)

		expect(saved.filter?.filter).toBe('project = 7 && labels = 14')
	})

	it('(j) keeps a single label id for a non-strict operator after an edit', async () => {
		const wrapper = mountForm(bugView())
		await loadLabels()

		inputs(wrapper)[0].vm.$emit('update:modelValue', 'labels like Bug')
		const saved = await save(wrapper)

		expect(saved.filter?.filter).toBe('labels like 14')
	})
})
