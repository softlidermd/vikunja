import {describe, it, expect, vi} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {h} from 'vue'

vi.mock('@/composables/useIsMobile', async importOriginal => ({
	...await importOriginal<typeof import('@/composables/useIsMobile')>(),
	useIsMobile: () => ({value: false}),
}))
vi.mock('vue-i18n', () => ({useI18n: () => ({t: (key: string) => key})}))

import TaskPropertyChip from './TaskPropertyChip.vue'

function mountChip(canWrite: boolean) {
	return mount(TaskPropertyChip, {
		attachTo: document.body,
		props: {label: 'Priority', canWrite},
		slots: {
			value: () => 'High',
			default: () => h('div', [h('button', {class: 'first-button'}), h('select', {class: 'editor'})]),
		},
		global: {
			directives: {tooltip: {}},
			stubs: {Icon: true},
			mocks: {$t: (key: string) => key},
		},
	})
}

describe('TaskPropertyChip', () => {
	it('focuses the editor field, not the first button, when opened', async () => {
		const wrapper = mountChip(true)
		;(wrapper.vm as unknown as {open: () => void}).open()
		await flushPromises()

		expect(document.activeElement?.classList.contains('editor')).toBe(true)
		wrapper.unmount()
	})

	it('emits remove from the × button', async () => {
		const wrapper = mountChip(true)
		await wrapper.find('.property-chip__remove').trigger('click')

		expect(wrapper.emitted('remove')).toHaveLength(1)
		wrapper.unmount()
	})

	it('is a plain chip without editor or × for read-only users', () => {
		const wrapper = mountChip(false)

		expect(wrapper.find('.property-chip__remove').exists()).toBe(false)
		expect(wrapper.find('.editor').exists()).toBe(false)
		expect(wrapper.text()).toContain('High')
		wrapper.unmount()
	})
})
