import type {Page} from '@playwright/test'

const ALWAYS_SHOWN_PROPERTIES: Record<string, string> = {
	'Assign to User': 'Assignees',
	'Add Labels': 'Labels',
	'Set Priority': 'Priority',
	'Set Effort': 'Effort',
	'Set Due Date': 'Due Date',
}

/**
 * Opens one of the task detail menus and clicks the item with the given name.
 * The menus mount eagerly, so their items sit hidden in the DOM; getByRole skips hidden ones.
 */
export async function openTaskAction(page: Page, menu: 'addProperty' | 'more', name: string) {
	// These fields are always-shown tiles now, not '+ More' items: open the tile instead.
	const tile = menu === 'addProperty' ? ALWAYS_SHOWN_PROPERTIES[name] : undefined
	if (tile) {
		return openTaskProperty(page, tile)
	}
	const dropdown = page.locator(`.task-view [data-cy="taskDetail.${menu}"]`)
	await dropdown.locator('.dropdown-trigger').first().click()
	await dropdown.getByRole('button', {name, exact: true}).click()
}

/**
 * Presses a task detail shortcut with nothing focused. Clicking the plain meta text blurs any
 * input first; pressing Esc instead would close the task dialog.
 */
export async function pressTaskShortcut(page: Page, key: string) {
	await page.locator('.task-view .heading p.created').click()
	await page.locator('body').press(key)
}

/**
 * Opens the popover of a property chip (e.g. 'Labels', 'Effort') so its editor can be used.
 */
export async function openTaskProperty(page: Page, name: string) {
	await page.locator('.task-view .properties .property').filter({hasText: name}).locator('.property-chip__body').click()
}
