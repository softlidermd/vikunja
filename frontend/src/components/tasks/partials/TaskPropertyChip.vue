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
				>
					<slot />
				</div>
			</template>
		</Popup>
		<span
			v-else
			class="property-chip__body"
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
}>()

const isOpen = ref(false)
const root = ref<HTMLElement | null>(null)
const trigger = ref<InstanceType<typeof BaseButton> | null>(null)
const triggerEl = computed<HTMLElement | null>(() => trigger.value?.$el ?? null)
const panel = ref<HTMLElement | null>(null)

watch(isOpen, async open => {
	await nextTick()
	if (open) {
		const field = panel.value?.querySelector<HTMLElement>('input:not([type="hidden"]), select, textarea')
			?? panel.value?.querySelector<HTMLElement>('button, [tabindex]:not([tabindex="-1"])')
		// On mobile a focused text field pops the keyboard over the sheet.
		if (field && (canAutoFocus() || !field.matches('input[type="text"], textarea'))) {
			field.focus()
		}
		return
	}
	// The hidden popover drops focus to the body; hand it back to the chip.
	if (!document.activeElement || document.activeElement === document.body) {
		triggerEl.value?.focus()
	}
})

function onChange() {
	if (!props.closeOnChange) {
		return
	}
	isOpen.value = false
	triggerEl.value?.focus()
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

.property-chip__label {
	color: var(--text-muted);
}

.property-chip__value {
	display: inline-flex;
	align-items: center;
	gap: .375rem;
	min-inline-size: 0;
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
