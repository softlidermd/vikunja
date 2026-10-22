<template>
	<span
		v-if="kanbanView && currentBucket && !isHiddenDefault"
		class="task-status"
		:class="{
			'is-done': currentBucket.id === kanbanView.done_bucket_id,
			'is-in-progress': progress === 'in-progress',
			'is-editable': canWrite,
		}"
	>
		<template v-if="canWrite">
			<select
				v-model="selected"
				:aria-label="$t('task.detail.bucketSelectLabel', {bucket: currentBucket.title})"
				@click.stop
				@change="changeBucket"
			>
				<option
					v-for="bucket in options"
					:key="bucket.id"
					:value="bucket.id"
				>
					{{ bucket.title }}
				</option>
			</select>
			<Icon
				icon="chevron-down"
				class="task-status__caret"
			/>
		</template>
		<template v-else>
			{{ currentBucket.title }}
		</template>
	</span>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'

import type {Task} from '@/client/generated'
import {isRequestContextAbort} from '@/client/requestContext'
import {useTaskStatus} from '@/composables/useTaskStatus'

const props = defineProps<{
	task: Task,
	canWrite: boolean,
	// Where most tasks sit, so a compact row only shows the other statuses.
	hideDefaultBucket?: boolean,
}>()

const {
	kanbanView,
	buckets,
	currentBucket,
	defaultBucketId,
	progress,
	moveToBucket,
} = useTaskStatus(() => props.task, () => props.canWrite)

const isHiddenDefault = computed(() => props.hideDefaultBucket && currentBucket.value?.id === defaultBucketId.value)
// Until the buckets load, the select still has to show the current one.
const options = computed(() => buckets.value.length
	? buckets.value
	: [currentBucket.value!])

const selected = ref<number>()
watch(() => currentBucket.value?.id, id => {
	selected.value = id
}, {immediate: true})

async function changeBucket() {
	const bucket = selected.value
	if (bucket === undefined) {
		return
	}
	let placed: number | undefined
	try {
		placed = await moveToBucket(bucket)
	} catch (e) {
		if (isRequestContextAbort(e)) {
			return
		}
		// The error toast is done by contextMutationOptions.
	}
	// Show where the task really is, unless the user already picked again.
	if (selected.value === bucket) {
		selected.value = placed ?? currentBucket.value?.id
	}
}
</script>

<style lang="scss" scoped>
.task-status {
	position: relative;
	display: inline-flex;
	align-items: center;
	max-inline-size: 12rem;
	padding: .0625rem .625rem;
	border-radius: 999px;
	background: var(--grey-100);
	color: var(--grey-700);
	font-size: .8rem;
	line-height: 1.375rem;
	white-space: nowrap;
	vertical-align: middle;

	&.is-in-progress {
		background: hsla(var(--primary-hsl), .2);
		color: var(--text);
	}

	&.is-done {
		// A tint with theme text: bright green text on a light background fails contrast.
		background: hsla(var(--success-h), var(--success-s), var(--success-l), .2);
		color: var(--text);
	}

	&.is-editable {
		padding: 0;

		&:hover {
			background: var(--grey-200);
		}

		&.is-in-progress:hover {
			background: hsla(var(--primary-hsl), .3);
		}

		&.is-done:hover {
			background: hsla(var(--success-h), var(--success-s), var(--success-l), .3);
		}
	}

	select {
		appearance: none;
		max-inline-size: 100%;
		padding-block: .125rem;
		padding-inline: .625rem 1.5rem;
		border: 0;
		border-radius: inherit;
		background: transparent;
		color: inherit;
		font: inherit;
		text-overflow: ellipsis;
		cursor: pointer;

		&:focus-visible {
			outline: 2px solid hsla(var(--primary-hsl), .5);
			outline-offset: 1px;
		}
	}

	&__caret {
		position: absolute;
		inset-inline-end: .625rem;
		font-size: .55rem;
		pointer-events: none;
	}
}
</style>
