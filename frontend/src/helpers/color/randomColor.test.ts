import {test, expect} from 'vitest'

import {colorIsDark} from './colorIsDark'
import {getRandomColorHex, COLORS} from './randomColor'

test('every random label color is pale enough for dark text', () => {
	for (const color of COLORS) {
		// colorIsDark is true when the background needs dark text, i.e. it is light.
		expect(colorIsDark(color), color).toBe(true)
	}
})

test('picks one of the palette colors', () => {
	expect(COLORS).toContain(getRandomColorHex())
})
