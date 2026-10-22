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
			@click="refreshApp()"
		>
			{{ $t('update.do') }}
		</XButton>
	</div>
</template>

<script lang="ts" setup>
import {computed, ref} from 'vue'
import {useRouter} from 'vue-router'
import {useBaseStore} from '@/stores/base'

const baseStore = useBaseStore()
const router = useRouter()

const updateAvailable = computed(() => baseStore.updateAvailable)
const registration = ref<ServiceWorkerRegistration | null>(null)
const refreshing = ref(false)
// Where to load once the new version took over; empty reloads the current page.
let reloadTo = ''

document.addEventListener('swUpdated', showRefreshUI, {once: true})

navigator?.serviceWorker?.addEventListener(
	'controllerchange', () => {
		// clientsClaim() also fires this on first install — only reload after
		// the user opted into an update, or unsaved state (e.g. login form
		// input) gets wiped.
		if (!refreshing.value) return
		refreshing.value = false
		if (reloadTo) {
			window.location.assign(reloadTo)
		} else {
			window.location.reload()
		}
	},
)

// Moving to another page is the safe moment to switch: that page loads fresh anyway, so no unsaved input is lost.
// Query changes (filters, sort) stay on the same page and do not count. The in-app navigation is cancelled
// and the page loaded by the new version: activating it deletes the old version's code the app would load next.
router.beforeEach((to, from) => {
	if (!updateAvailable.value || to.path === from.path || !registration.value?.waiting) {
		return
	}
	reloadTo = router.resolve(to.fullPath).href
	refreshApp()
	return false
})

function showRefreshUI(e: Event) {
	console.log('recieved refresh event', e)
	const customEvent = e as CustomEvent<ServiceWorkerRegistration>
	registration.value = customEvent.detail
	baseStore.setUpdateAvailable(true)
}

function refreshApp() {
	baseStore.setUpdateAvailable(false)
	if (!registration.value || !registration.value.waiting) {
		return
	}
	refreshing.value = true
	// Notify the service worker to actually do the update
	registration.value.waiting.postMessage('skipWaiting')
}
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
