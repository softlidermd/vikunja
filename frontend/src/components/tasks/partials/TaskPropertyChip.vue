<template>
	<div
		ref="root"
		class="property-chip"
		:class="{'is-open': isOpen}"
	>
		<Popup
			v-if="$slots.default && canWrite"
			v-model:open="isOpen"
			placement="bottom-start"
			:anchor="root"
			sheet-on-mobile
			:sheet-title="label"
		>
			<template #trigger="{toggle}">
				<BaseButton
					ref="trigger"
					v-shortcut="shortcut"
					class="property-chip__body"
					:aria-expanded="isOpen"
					@click="toggle()"
				>
					<span class="property-chip__label">{{ label }}</span>
					<span class="property-chip__value"><slot name="value" /></span>
				</BaseButton>
			</template>
			<template #content>
				<div
					ref="panel"
					class="property-chip__popover"
					role="dialog"
					:aria-label="label"
					@change="onChange"
					@keydown="onKeydown"
					@pointerdown="keyedSelect = false"
				>
					<slot />
				</div>
			</template>
		</Popup>
		<!-- Editors with their own trigger (the datepicker): a click on the label opens them too. -->
		<span
			v-else
			v-shortcut="canWrite ? shortcut : ''"
			class="property-chip__body"
			:class="{'is-clickable': canWrite}"
			@click="canWrite && emit('activate')"
		>
			<span class="property-chip__label">{{ label }}</span>
			<span class="property-chip__value"><slot name="value" /></span>
		</span>
		<BaseButton
			v-if="canWrite && removable"
			v-tooltip="removeLabel || $t('task.detail.removeProperty', {property: label})"
			class="property-chip__remove"
			:aria-label="removeLabel || $t('task.detail.removeProperty', {property: label})"
			@click="emit('remove')"
		>
			<Icon icon="times" />
		</BaseButton>
	</div>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, watch} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import Popup from '@/components/misc/Popup.vue'
import {canAutoFocus} from '@/directives/focus'

const props = withDefaults(defineProps<{
	label: string
	canWrite: boolean
	removeLabel?: string
	// For single-value editors (selects, colour): picking a value is the whole job.
	closeOnChange?: boolean
	// Off for the always-shown tiles while they hold no value: there is nothing to clear.
	removable?: boolean
	shortcut?: string
}>(), {
	removeLabel: '',
	closeOnChange: false,
	removable: true,
	shortcut: '',
})

const emit = defineEmits<{
	remove: []
	activate: []
}>()

const isOpen = ref(false)
const root = ref<HTMLElement | null>(null)
const trigger = ref<InstanceType<typeof BaseButton> | null>(null)
const triggerEl = computed<HTMLElement | null>(() => trigger.value?.$el ?? null)
const panel = ref<HTMLElement | null>(null)

// Sync, so a close still sees where focus was before the popover hid.
watch(isOpen, open => {
	if (open) {
		nextTick(() => {
			const field = panel.value?.querySelector<HTMLElement>('input:not([type="hidden"]), select, textarea')
				?? panel.value?.querySelector<HTMLElement>('button, [tabindex]:not([tabindex="-1"])')
			// On mobile a focused text field pops the keyboard over the sheet.
			if (field && (canAutoFocus() || !field.matches('input[type="text"], textarea'))) {
				field.focus()
			}
		})
		return
	}
	// Hiding the popover would drop focus to the body; hand it back to the chip.
	if (panel.value?.contains(document.activeElement)) {
		nextTick(() => triggerEl.value?.focus({preventScroll: true}))
	}
}, {flush: 'sync'})

// Chrome and Firefox on Windows fire `change` on every arrow press on a closed select, so after keyboard
// browsing the popover waits for Enter instead of closing on the first step.
let keyedSelect = false
const SELECT_BROWSE_KEYS = ['ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown']

function onKeydown(event: KeyboardEvent) {
	if (!(event.target instanceof HTMLSelectElement)) {
		keyedSelect = false
		return
	}
	if (event.key === 'Enter') {
		keyedSelect = false
		if (props.closeOnChange) {
			event.preventDefault()
			closeAndFocusChip()
		}
		return
	}
	keyedSelect = SELECT_BROWSE_KEYS.includes(event.key) || event.key.length === 1
}

function onChange() {
	if (props.closeOnChange && !keyedSelect) {
		closeAndFocusChip()
	}
}

// A picked value finishes the job, so go back to the chip even if focus had wandered off the select.
function closeAndFocusChip() {
	isOpen.value = false
	nextTick(() => triggerEl.value?.focus({preventScroll: true}))
}

defineExpose({
	open: () => {
		isOpen.value = true
	},
})
</script>

<style lang="scss" scoped>
// A tile: the label above its value, so a grid of them reads like a form.
.property-chip {
	position: relative;
	display: flex;
	min-inline-size: 0;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	background: var(--grey-50);
	font-size: .8125rem;
	line-height: 1.3;
	color: var(--text);
	transition: border-color $transition;

	&:hover,
	&.is-open {
		border-color: var(--grey-300);
	}
}

.property-chip__body {
	display: flex;
	flex: 1;
	flex-direction: column;
	align-items: flex-start;
	gap: .125rem;
	min-inline-size: 0;
	padding: .375rem .625rem;
	color: inherit;
	text-align: start;
	white-space: nowrap;

	.property-chip:has(.property-chip__remove) & {
		padding-inline-end: 1.75rem;
	}
}

.property-chip__body.is-clickable {
	cursor: pointer;
}

.property-chip__label {
	font-size: .75rem;
	color: var(--text-muted);
}

.property-chip__value {
	display: flex;
	align-items: center;
	gap: .375rem;
	max-inline-size: 100%;
	min-block-size: 1.25rem;
	// Room for a focused value's (the datepicker trigger's) focus ring inside the clip.
	margin: -2px;
	padding: 2px;
	overflow: hidden;
	font-size: .875rem;
	font-weight: 600;
}

.property-chip__remove {
	position: absolute;
	inset-block-start: .25rem;
	inset-inline-end: .25rem;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	inline-size: 1.375rem;
	block-size: 1.375rem;
	border-radius: $radius;
	font-size: .6875rem;
	color: var(--text-muted);

	&:hover,
	&:focus-visible {
		background: var(--grey-200);
		color: var(--text);
	}
}

.property-chip__popover {
	max-inline-size: calc(100vw - 1rem);
	padding: .75rem;
	border: 1px solid var(--grey-200);
	border-radius: $radius-large;
	background: var(--white);
	box-shadow: var(--shadow-md);
	font-size: .875rem;
	line-height: 1.5;

	.bottom-sheet & {
		inline-size: 100%;
		max-inline-size: none;
		border: 0;
		box-shadow: none;
	}
}
</style>
