import {register} from 'register-service-worker'

import router from './router'
import {useBaseStore} from './stores/base'
import {getFullBaseUrl} from './helpers/getFullBaseUrl'

const UPDATE_CHECK_INTERVAL = 15 * 60 * 1000
// The new worker takes over within moments; if it never does (a newer deploy replaced it), load anyway.
const SWITCH_TIMEOUT = 3000

let updateRegistration: ServiceWorkerRegistration | undefined
let switching = false
let loading = false
let target = ''

function markUpdateAvailable(registration: ServiceWorkerRegistration) {
	updateRegistration = registration
	useBaseStore().setUpdateAvailable(true)
}

function load() {
	if (loading) return
	loading = true
	if (target) {
		window.location.assign(target)
	} else {
		window.location.reload()
	}
}

// Loads `href`, or the current page, in the new version.
export function applyUpdate(href = '') {
	if (switching) return
	switching = true
	target = href
	const waiting = updateRegistration?.waiting
	// Nothing waiting: another tab already switched, so the server and the worker both have the new version.
	if (!waiting) {
		load()
		return
	}
	waiting.postMessage('skipWaiting')
	setTimeout(load, SWITCH_TIMEOUT)
}

function checkForUpdate(registration: ServiceWorkerRegistration) {
	// Rejects while the server is unreachable; the next check retries.
	registration.update().catch(() => {})
}

if (import.meta.env.PROD) {
	// clientsClaim() also fires this in other tabs and on first install: only the tab that asked for the switch loads.
	navigator.serviceWorker?.addEventListener('controllerchange', () => {
		if (switching) load()
	})

	// Moving to another page is the safe moment to switch: that page loads fresh anyway, so no unsaved input is lost.
	// Query changes (filters, sort) stay on the same page. The in-app navigation is cancelled and the page loaded by
	// the new version, because activating it deletes the old version's code the app would load next.
	router.beforeEach((to, from) => {
		if (!useBaseStore().updateAvailable || to.path === from.path) {
			return
		}
		applyUpdate(router.resolve(to.fullPath).href)
		return false
	})

	register(getFullBaseUrl() + 'sw.js', {
		ready() {
			console.log('App is being served from cache by a service worker.')
		},
		registered(registration) {
			console.log('Service worker has been registered.')
			// The browser looks for a new version only on a full page load, which a tab open all day never does.
			setInterval(() => checkForUpdate(registration), UPDATE_CHECK_INTERVAL)
			document.addEventListener('visibilitychange', () => {
				if (document.visibilityState === 'visible') {
					checkForUpdate(registration)
				}
			})
			// The library reports only the first update of a page that loaded with one already waiting.
			registration.addEventListener('updatefound', () => {
				const worker = registration.installing
				worker?.addEventListener('statechange', () => {
					if (worker.state === 'installed' && navigator.serviceWorker.controller) {
						markUpdateAvailable(registration)
					}
				})
			})
		},
		cached() {
			console.log('Content has been cached for offline use.')
		},
		updatefound() {
			console.log('New content is downloading.')
		},
		updated(registration) {
			console.log('New content is available; please refresh.')
			markUpdateAvailable(registration)
		},
		offline() {
			console.log('No internet connection found. App is running in offline mode.')
		},
		error(error) {
			console.error('Error during service worker registration:', error)
		},
	})
}
