<template>
	<div class="select">
		<!-- Colored on the select itself: Bulma paints the select, not its wrapper. -->
		<select
			v-model="priority"
			class="priority-chip"
			:data-priority="priority"
			:disabled="disabled || undefined"
			:aria-label="$t('task.attributes.priority')"
		>
			<option :value="PRIORITIES.UNSET">
				{{ $t('task.priority.unset') }}
			</option>
			<option :value="PRIORITIES.LOW">
				{{ $t('task.priority.low') }}
			</option>
			<option :value="PRIORITIES.MEDIUM">
				{{ $t('task.priority.medium') }}
			</option>
			<option :value="PRIORITIES.HIGH">
				{{ $t('task.priority.high') }}
			</option>
			<option :value="PRIORITIES.URGENT">
				{{ $t('task.priority.urgent') }}
			</option>
			<option :value="PRIORITIES.DO_NOW">
				{{ $t('task.priority.doNow') }}
			</option>
		</select>
	</div>
</template>

<script setup lang="ts">
import {PRIORITIES} from '@/constants/priorities'

withDefaults(defineProps<{
	disabled?: boolean
}>(), {
	disabled: false,
})

const priority = defineModel<number>({
	required: true,
	default: 0,
})

</script>
