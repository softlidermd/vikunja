import {describe, it, expect, vi} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {h} from 'vue'

vi.mock('@/composables/useIsMobile', async importOriginal => ({
	...await importOriginal<typeof import('@/composables/useIsMobile')>(),
	useIsMobile: () => ({value: false}),
}))
vi.mock('vue-i18n', () => ({useI18n: () => ({t: (key: string) => key})}))

import TaskPropertyChip from './TaskPropertyChip.vue'

function mountChip(canWrite: boolean, closeOnChange = false) {
	return mount(TaskPropertyChip, {
		attachTo: document.body,
		props: {label: 'Priority', canWrite, closeOnChange},
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

	it('closes after a mouse pick and hands focus back to the chip', async () => {
		const wrapper = mountChip(true, true)
		;(wrapper.vm as unknown as {open: () => void}).open()
		await flushPromises()
		const select = wrapper.find('select.editor')
		const trigger = wrapper.find('.property-chip__body')
		expect(trigger.attributes('aria-expanded')).toBe('true')

		await select.trigger('pointerdown')
		await select.trigger('change')
		await flushPromises()

		expect(trigger.attributes('aria-expanded')).toBe('false')
		expect(document.activeElement).toBe(trigger.element)
		wrapper.unmount()
	})

	it('stays open while browsing the select with the keyboard and closes on Enter', async () => {
		const wrapper = mountChip(true, true)
		;(wrapper.vm as unknown as {open: () => void}).open()
		await flushPromises()
		const select = wrapper.find('select.editor')
		const trigger = wrapper.find('.property-chip__body')

		await select.trigger('keydown', {key: 'ArrowDown'})
		await select.trigger('change')
		await flushPromises()
		expect(trigger.attributes('aria-expanded')).toBe('true')

		await select.trigger('keydown', {key: 'Enter'})
		await new Promise(resolve => setTimeout(resolve))
		await flushPromises()
		expect(trigger.attributes('aria-expanded')).toBe('false')
		expect(document.activeElement).toBe(trigger.element)
		wrapper.unmount()
	})

	it.each([true, false])('opens the select list right away only for single-value chips (closeOnChange %s)', async closeOnChange => {
		const showPicker = vi.fn()
		Object.defineProperty(HTMLSelectElement.prototype, 'showPicker', {value: showPicker, configurable: true})
		vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
			cb(0)
			return 0
		})
		const wrapper = mountChip(true, closeOnChange)
		;(wrapper.vm as unknown as {open: () => void}).open()
		await flushPromises()

		expect(showPicker).toHaveBeenCalledTimes(closeOnChange ? 1 : 0)
		wrapper.unmount()
		vi.unstubAllGlobals()
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
