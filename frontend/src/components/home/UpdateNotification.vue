<template>
	<div
		v-if="updateAvailable"
		class="update-notification"
	>
		<p class="update-notification__message">
			{{ $t('update.available') }}
		</p>
		<XButton
			:shadow="false"
			:wrap="false"
			@click="applyUpdate()"
		>
			{{ $t('update.do') }}
		</XButton>
	</div>
</template>

<script lang="ts" setup>
import {computed} from 'vue'
import {useBaseStore} from '@/stores/base'
import {applyUpdate} from '@/registerServiceWorker'

const baseStore = useBaseStore()

const updateAvailable = computed(() => baseStore.updateAvailable)
</script>

<style lang="scss" scoped>
.update-notification {
	position: fixed;
	// FIXME: We should prevent usage of z-index or
	// at least define it centrally
	// the highest z-index of a modal is .hint-modal with 4500
	z-index: 5000;
	inset-block-end: 1rem;
	inset-inline: 1rem;
	max-inline-size: max-content;
	margin-inline: auto;

	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	padding: .5rem .5rem .5rem 1rem;
	background: $warning;
	border-radius: $radius;
	font-size: .9rem;
	color: hsl(220.9, 39.3%, 11%); // color copied to avoid it changing in dark mode
}

.update-notification__message {
	inline-size: 100%;
	text-align: center;
}
</style>
