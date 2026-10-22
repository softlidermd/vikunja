import {register} from 'register-service-worker'

import {getFullBaseUrl} from './helpers/getFullBaseUrl'

const UPDATE_CHECK_INTERVAL = 15 * 60 * 1000

if (import.meta.env.PROD) {
	register(getFullBaseUrl() + 'sw.js', {
		ready() {
			console.log('App is being served from cache by a service worker.')
		},
		registered(registration) {
			console.log('Service worker has been registered.')
			// The browser looks for a new version only on a full page load, which a tab open all day never does.
			setInterval(() => registration.update(), UPDATE_CHECK_INTERVAL)
			document.addEventListener('visibilitychange', () => {
				if (document.visibilityState === 'visible') {
					registration.update()
				}
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
			// Send an event with the updated info
			document.dispatchEvent(
				new CustomEvent('swUpdated', {detail: registration}),
			)
		},
		offline() {
			console.log('No internet connection found. App is running in offline mode.')
		},
		error(error) {
			console.error('Error during service worker registration:', error)
		},
	})
}
