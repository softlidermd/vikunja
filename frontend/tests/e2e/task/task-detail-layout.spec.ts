import {test, expect} from '../../support/fixtures'
import {pressTaskShortcut} from '../../support/taskDetail'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {createDefaultViews} from '../project/prepareProjects'
import type {Page} from '@playwright/test'

async function openTitleOnlyTask(page: Page) {
	const [project] = await ProjectFactory.create(1)
	await createDefaultViews(project.id)
	const [task] = await TaskFactory.create(1, {id: 1, project_id: project.id, done: false})
	await page.goto(`/tasks/${task.id}`)
	await expect(page.locator('.task-view .heading p.created')).toBeVisible({timeout: 10000})
	return task
}

const property = (page: Page, title: string) => page.locator('.task-view .properties .property').filter({hasText: title})

test.describe('Task detail layout', () => {
	test('opens every field and section with its shortcut while the menus are closed', async ({authenticatedPage: page}) => {
		await openTitleOnlyTask(page)

		const steps = [
			['KeyL', page.locator('.task-view .property.labels-list')],
			['KeyP', property(page, 'Priority')],
			['KeyC', property(page, 'Color')],
			['KeyA', page.locator('.task-view .property.assignees')],
			['KeyD', property(page, 'Due Date')],
			['Alt+KeyR', property(page, 'Reminders')],
			['KeyM', page.locator('.task-view h2').filter({hasText: 'Move task to a different project'})],
			['KeyR', page.locator('.task-view .task-relations')],
			['KeyE', page.locator('.task-view .details.content.description .editor-toolbar')],
		] as const

		for (const [key, target] of steps) {
			await expect(target, `${key} target before the key press`).toBeHidden()
			await pressTaskShortcut(page, key)
			await expect(target, `${key} target after the key press`).toBeVisible()
		}
	})

	test('marks the task done with t, and it stays done after a reload', async ({authenticatedPage: page, apiContext, userToken}) => {
		const task = await openTitleOnlyTask(page)
		const done = page.locator('.task-view [data-cy="taskDetail.done"]')
		await expect(done).toHaveAttribute('aria-pressed', 'false')

		await pressTaskShortcut(page, 'KeyT')
		await expect.poll(async () => {
			const stored = await apiContext.get(`tasks/${task.id}`, {headers: {Authorization: `Bearer ${userToken}`}})
			return (await stored.json()).done
		}).toBe(true)
		await page.reload()
		await expect(done).toHaveAttribute('aria-pressed', 'true')
	})

	test('favorites the task with s, and it stays a favorite after a reload', async ({authenticatedPage: page, apiContext, userToken}) => {
		const task = await openTitleOnlyTask(page)
		const star = page.locator('.task-view .task-topbar__actions [aria-pressed]').first()
		await expect(star).toHaveAttribute('aria-pressed', 'false')

		await pressTaskShortcut(page, 'KeyS')
		await expect.poll(async () => {
			const stored = await apiContext.get(`tasks/${task.id}`, {headers: {Authorization: `Bearer ${userToken}`}})
			return (await stored.json()).is_favorite
		}).toBe(true)
		await page.reload()
		await expect(star).toHaveAttribute('aria-pressed', 'true')
	})

	test('opens the file chooser with f', async ({authenticatedPage: page}) => {
		await openTitleOnlyTask(page)
		const fileChooser = page.waitForEvent('filechooser')
		await pressTaskShortcut(page, 'KeyF')
		await fileChooser
	})

	test('asks to delete the task with the Delete key', async ({authenticatedPage: page}) => {
		await openTitleOnlyTask(page)
		await pressTaskShortcut(page, 'Delete')
		await expect(page.locator('dialog[open] .modal-content .modal-header')).toContainText('Delete this task')
	})

	test('Esc closes an open ⋯ menu and keeps the task dialog open', async ({authenticatedPage: page}) => {
		const task = await openTitleOnlyTask(page)
		const more = page.locator('.task-view [data-cy="taskDetail.more"]')
		await more.locator('.dropdown-trigger').click()
		const deleteItem = more.getByRole('button', {name: 'Delete', exact: true})
		await expect(deleteItem).toBeVisible()

		await page.keyboard.press('Escape')
		await expect(deleteItem).toBeHidden()
		await expect(more.locator('.dropdown-trigger')).toBeFocused()
		// Closing the dialog navigates away; give that a moment to happen if Esc leaked through.
		await page.waitForTimeout(500)
		await expect(page.locator('.task-view')).toBeVisible()
		await expect(page).toHaveURL(new RegExp(`/tasks/${task.id}`))
	})

	test('the last + Property item can be scrolled to when the menu is taller than the space below', async ({authenticatedPage: page}) => {
		await openTitleOnlyTask(page)
		const addProperty = page.locator('.task-view [data-cy="taskDetail.addProperty"]')
		await addProperty.locator('.dropdown-trigger').click()
		const lastItem = addProperty.getByRole('button', {name: 'Set Color', exact: true})
		// The overlay scrolls instead of clipping the menu.
		await lastItem.scrollIntoViewIfNeeded()
		await expect(lastItem).toBeInViewport({ratio: 1})
		await lastItem.click()
		await expect(property(page, 'Color')).toBeVisible()
	})

	test.describe('at 1920×1080', () => {
		test.use({viewport: {width: 1920, height: 1080}})

		test('a title-only task is a compact card without a toolbar or action grid', async ({authenticatedPage: page}) => {
			await openTitleOnlyTask(page)
			await expect(page.locator('.task-view .tiptap__collapsed')).toHaveCount(2)

			await expect(page.locator('.task-view .editor-toolbar:visible')).toHaveCount(0)
			await expect(page.locator('.task-view .action-buttons')).toHaveCount(0)
			const card = await page.locator('.task-detail-view-modal .modal-content').boundingBox()
			expect(card?.height).toBeLessThanOrEqual(540)
		})

		test('shows the last + Property item and the label search results unclipped on a title-only task', async ({authenticatedPage: page}) => {
			await openTitleOnlyTask(page)
			const addProperty = page.locator('.task-view [data-cy="taskDetail.addProperty"]')
			await addProperty.locator('.dropdown-trigger').click()
			const lastItem = addProperty.getByRole('button', {name: 'Set Color', exact: true})
			await expect(lastItem).toBeInViewport({ratio: 1})
			await lastItem.click()
			await expect(property(page, 'Color')).toBeVisible()

			await pressTaskShortcut(page, 'KeyL')
			await page.locator('.task-view .labels-list .multiselect input').fill('new label')
			const results = page.locator('.task-view .labels-list .multiselect .search-results')
			await expect(results).toBeVisible()
			await expect(results).toBeInViewport({ratio: 1})
		})
	})
})
