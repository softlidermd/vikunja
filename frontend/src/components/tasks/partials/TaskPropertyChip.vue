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
			class="property-chip__body"
			:class="{'is-clickable': canWrite}"
			@click="canWrite && emit('activate')"
		>
			<span class="property-chip__label">{{ label }}</span>
			<span class="property-chip__value"><slot name="value" /></span>
		</span>
		<BaseButton
			v-if="canWrite"
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
}>(), {
	removeLabel: '',
	closeOnChange: false,
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
.property-chip {
	display: inline-flex;
	align-items: center;
	gap: .125rem;
	max-inline-size: 100%;
	block-size: 1.875rem;
	padding-inline: .625rem .25rem;
	border: 1px solid var(--grey-200);
	border-radius: $radius;
	background: var(--grey-50);
	font-size: .8125rem;
	line-height: 1;
	color: var(--text);
	transition: border-color $transition;

	&:hover,
	&.is-open {
		border-color: var(--grey-300);
	}

	// Read-only chips have no remove button, so balance the padding.
	&:not(:has(.property-chip__remove)) {
		padding-inline-end: .625rem;
	}
}

.property-chip__body {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	min-inline-size: 0;
	block-size: 100%;
	color: inherit;
	white-space: nowrap;
}

.property-chip__body.is-clickable {
	cursor: pointer;
}

.property-chip__label {
	color: var(--text-muted);
}

.property-chip__value {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	min-inline-size: 0;
	// Room for a focused value's (the datepicker trigger's) focus ring inside the clip.
	margin: -2px;
	padding: 2px;
	overflow: hidden;
	font-weight: 600;

	&:empty {
		display: none;
	}
}

.property-chip__remove {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	flex: none;
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
