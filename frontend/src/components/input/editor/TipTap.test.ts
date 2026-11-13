import {describe, it, expect, vi, afterEach} from 'vitest'
import {flushPromises, mount, type VueWrapper} from '@vue/test-utils'
import {createPinia} from 'pinia'
import BaseButton from '@/components/base/BaseButton.vue'
import XButton from '@/components/input/Button.vue'

vi.mock('vue-i18n', async importOriginal => ({
	...(await importOriginal<typeof import('vue-i18n')>()),
	useI18n: () => ({t: (key: string) => key}),
}))

import TipTap from './TipTap.vue'
import {saveEditorDraft} from '@/helpers/editorDraftStorage'

const mounted: VueWrapper[] = []

async function mountEditor(props: Record<string, unknown>) {
	const wrapper = mount(TipTap, {
		attachTo: document.body,
		props: {placeholder: 'Add a description…', ...props},
		global: {
			plugins: [createPinia()],
			components: {XButton, BaseButton},
			stubs: {Icon: true, EditorToolbar: true, BubbleMenu: true, ImageLightbox: true},
			directives: {tooltip: {}, cy: {}},
			mocks: {$t: (key: string) => key},
		},
	})
	mounted.push(wrapper)
	await flushPromises()
	return wrapper
}

afterEach(() => {
	mounted.splice(0).forEach(w => w.unmount())
	localStorage.clear()
})

describe('TipTap collapseWhenEmpty', () => {
	it('keeps the old behaviour by default: an empty editor opens in edit mode', async () => {
		const wrapper = await mountEditor({modelValue: ''})
		expect(wrapper.find('.tiptap__collapsed').exists()).toBe(false)
		expect(wrapper.find('.tiptap__wrapper-is-editing').exists()).toBe(true)
	})

	it('renders a one-line button for empty content and expands on click', async () => {
		const wrapper = await mountEditor({modelValue: '<p></p>', collapseWhenEmpty: true})
		const collapsed = wrapper.find('button.tiptap__collapsed')
		expect(collapsed.text()).toBe('Add a description…')
		expect((wrapper.find('.tiptap__wrapper').element as HTMLElement).style.display).toBe('none')
		expect(wrapper.find('.done-edit').exists()).toBe(false)

		await collapsed.trigger('click')
		expect(wrapper.find('.tiptap__collapsed').exists()).toBe(false)
		expect(wrapper.find('.tiptap__wrapper-is-editing').exists()).toBe(true)
	})

	it('does not collapse non-empty content', async () => {
		const wrapper = await mountEditor({modelValue: '<p>hi</p>', collapseWhenEmpty: true})
		expect(wrapper.find('.tiptap__collapsed').exists()).toBe(false)
	})

	it('opens a stored draft expanded', async () => {
		saveEditorDraft('draft-key', '<p>draft</p>')
		const wrapper = await mountEditor({modelValue: '', collapseWhenEmpty: true, storageKey: 'draft-key'})
		expect(wrapper.find('.tiptap__collapsed').exists()).toBe(false)
		expect(wrapper.find('.tiptap__wrapper-is-editing').exists()).toBe(true)
	})

	it('Cancel discards the edit and emits discard', async () => {
		const wrapper = await mountEditor({
			modelValue: '',
			collapseWhenEmpty: true,
			showSave: true,
			enableDiscardShortcut: true,
		})
		await wrapper.find('.tiptap__collapsed').trigger('click')

		const buttons = wrapper.findAll('.tiptap__save-actions button')
		expect(buttons[1]!.text()).toBe('misc.cancel')
		await buttons[1]!.trigger('click')
		expect(wrapper.emitted('discard')).toHaveLength(1)
		expect(wrapper.emitted('update:modelValue')).toBeUndefined()
		expect(wrapper.find('.tiptap__collapsed').exists()).toBe(true)
	})

	it('has no Cancel button without enableDiscardShortcut', async () => {
		const wrapper = await mountEditor({modelValue: '', showSave: true})
		expect(wrapper.findAll('.tiptap__save-actions button')).toHaveLength(1)
	})
})
