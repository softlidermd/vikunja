import {error} from '@/message'
import {useI18n} from 'vue-i18n'

export function useCopyToClipboard() {
	const {t} = useI18n({useScope: 'global'})
	
	function fallbackCopyTextToClipboard(text: string): boolean {
		const textArea = document.createElement('textarea')
		textArea.value = text
		
		// Avoid scrolling to bottom
		textArea.style.top = '0'
		textArea.style.left = '0'
		textArea.style.position = 'fixed'
	
		document.body.appendChild(textArea)
		textArea.focus()
		textArea.select()
	
		let copied = true
		try {
			// NOTE: the execCommand is deprecated but as of 2022_09
			// widely supported and works without https
			const successful = document.execCommand('copy')
			if (!successful) {
				throw new Error()
			}
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
		} catch (e) {
			error(t('misc.copyError'))
			copied = false
		}
	
		document.body.removeChild(textArea)
		return copied
	}
	
	// Resolves to whether the text was copied; a failure has already shown an error.
	return async (text: string): Promise<boolean> => {
		if (!navigator.clipboard) {
			return fallbackCopyTextToClipboard(text)
		}
		try {
			await navigator.clipboard.writeText(text)
			return true
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
		} catch(e) {
			error(t('misc.copyError'))
			return false
		}
	}
}
