<template>
	<div class="select">
		<select
			v-model="size"
			:disabled="disabled || undefined"
			:aria-label="$t('task.attributes.size')"
		>
			<option
				v-for="value in TASK_SIZES"
				:key="value"
				:value="value"
			>
				{{ $t(taskSizeLabelKey(value)) }}
			</option>
		</select>
	</div>
</template>

<script setup lang="ts">
import {TASK_SIZES, taskSizeLabelKey, type TaskSize} from '@/constants/taskSizes'

withDefaults(defineProps<{
	disabled?: boolean
}>(), {
	disabled: false,
})

const size = defineModel<TaskSize>({
	required: true,
	default: 0,
})
</script>
