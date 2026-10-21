import type {Directive} from 'vue'

import {BULMA_MOBILE_BREAKPOINT} from '@/composables/useIsMobile'

const FOCUSABLE_TAGS = ['INPUT', 'SELECT', 'TEXTAREA']

function isFocusable(el: HTMLElement): boolean {
	return FOCUSABLE_TAGS.includes(el.tagName) || el.isContentEditable
}

function getFocusableElement(el: HTMLElement): HTMLElement | null {
	if (isFocusable(el)) {
		return el
	}
	// Look for the first focusable child
	return el.querySelector<HTMLElement>('input, select, textarea, [contenteditable="true"]')
}

// Auto focusing on mobile pops up the keyboard over half the screen.
export const canAutoFocus = () => window.innerWidth > BULMA_MOBILE_BREAKPOINT

const focus = <Directive<HTMLElement, string>>{
	// When the bound element is inserted into the DOM...
	mounted(el, {modifiers}) {
		if (canAutoFocus() || modifiers?.always) {
			const target = getFocusableElement(el)
			target?.focus()
		}
	},
}

export default focus
