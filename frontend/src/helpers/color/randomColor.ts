// Pale pastels, so auto-created labels look like the hand-picked ones and keep dark, readable text.
export const COLORS = [
	'#fecaca',
	'#fed7aa',
	'#fde68a',
	'#d9f99d',
	'#bbf7d0',
	'#99f6e4',
	'#bae6fd',
	'#c7d2fe',
	'#ddd6fe',
	'#f5d0fe',
	'#fbcfe8',
	'#e5e7eb',
]

export function getRandomColorHex(): string {
	return COLORS[Math.floor(Math.random() * COLORS.length)]
}
