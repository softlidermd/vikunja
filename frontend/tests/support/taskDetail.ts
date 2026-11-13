import type {Page} from '@playwright/test'

/**
 * Opens one of the task detail menus and clicks the item with the given name.
 * The menus mount eagerly, so their items sit hidden in the DOM; getByRole skips hidden ones.
 */
export async function openTaskAction(page: Page, menu: 'addProperty' | 'more', name: string) {
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
