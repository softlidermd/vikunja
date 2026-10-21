import type {Page} from '@playwright/test'
import {test, expect} from '../../support/fixtures'
import {TaskFactory} from '../../factories/task'
import {ProjectFactory} from '../../factories/project'
import {LabelFactory} from '../../factories/labels'
import {createDefaultViews, createProjects} from './prepareProjects'
import {createTasksWithPriorities, createTasksWithSearch} from '../../support/filterTestHelpers'

// Priority is hidden by default.
async function showPriorityColumn(page: Page) {
	const columns = page.getByRole('button', {
		name: 'Columns',
		exact: true,
	})
	await columns.click()
	await page.locator('.columns-filter').getByText('Priority', {
		exact: true,
	}).click()
	await columns.click()
}

test.describe('Project View Table', () => {
	test('Should show a table with tasks', async ({authenticatedPage: page}) => {
		await createProjects(1)
		const tasks = await TaskFactory.create(1, {
			project_id: 1,
		})
		await page.goto('/projects/1/3')

		await expect(page.locator('.project-table table.table')).toBeVisible()
		await expect(page.locator('.project-table table.table')).toContainText(tasks[0].title)
	})

	test('Should have working column switches', async ({authenticatedPage: page}) => {
		await createProjects(1)
		await TaskFactory.create(1, {
			project_id: 1,
		})
		const loadTasksPromise = page.waitForResponse(response =>
			response.url().includes('/projects/1/views/') && response.url().includes('/tasks'),
		)
		await page.goto('/projects/1/3')
		await loadTasksPromise

		// Click the Columns button to open the column selector
		await page.locator('.project-table .filter-container .button').filter({hasText: 'Columns'}).click()

		// Click Priority checkbox to enable Priority column (click on the text like Cypress does)
		await page.locator('.project-table .filter-container .card.columns-filter .card-content').getByText('Priority').click()

		// Wait for Priority checkbox to be checked
		await expect(page.getByRole('checkbox', {name: 'Checkbox Priority'})).toBeChecked()

		// Click Done checkbox to disable Done column (click on the text like Cypress does)
		await page.locator('.project-table .filter-container .card.columns-filter .card-content').getByText('Done', {exact: true}).click()

		// Wait for Done checkbox to be unchecked
		await expect(page.getByRole('checkbox', {name: 'Checkbox Done', exact: true})).not.toBeChecked()

		// Verify Priority column is now visible
		await expect(page.locator('.project-table table.table th').filter({hasText: 'Priority'})).toBeVisible()
		// Verify Done column is now hidden
		await expect(page.locator('.project-table table.table th').filter({hasText: /^Done$/})).not.toBeVisible()
	})

	test('Should show every priority for open and completed tasks', async ({authenticatedPage: page}) => {
		await createProjects(1)
		const priorities = ['Unset', 'Low', 'Medium', 'High', 'Urgent', 'DO NOW']
		const tasks = await TaskFactory.create(priorities.length * 2, {
			project_id: 1,
			title: (i: number) => `Priority task ${i}`,
			priority: (i: number) => (i - 1) % priorities.length,
			done: (i: number) => i > priorities.length,
		})
		await page.goto('/projects/1/3')
		await showPriorityColumn(page)

		for (const task of tasks) {
			const row = page.locator('.project-table tbody tr').filter({
				has: page.getByRole('link', {name: task.title, exact: true}),
			})
			await expect(row).toBeVisible()
			// Writers get an inline priority select instead of the label.
			await expect(row.getByRole('combobox', {name: 'Priority'})).toHaveValue(String(task.priority))
		}
	})

	test('Should edit priority and done inline and persist them', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await createProjects(1)
		const [task] = await TaskFactory.create(1, {
			project_id: 1,
			priority: 0,
		})
		await page.goto('/projects/1/3')
		await showPriorityColumn(page)
		const row = page.locator('.project-table tbody tr').filter({
			has: page.getByRole('link', {name: task.title, exact: true}),
		})
		const patched = () => page.waitForResponse(response =>
			response.request().method() === 'PATCH' && response.url().includes(`/tasks/${task.id}`),
		)

		let saved = patched()
		await row.getByRole('combobox', {name: 'Priority'}).selectOption('3')
		expect((await saved).ok()).toBeTruthy()

		saved = patched()
		await row.locator('.fancy-checkbox').click()
		expect((await saved).ok()).toBeTruthy()

		await page.reload()
		await expect(row.getByRole('combobox', {name: 'Priority'})).toHaveValue('3')
		await expect(row.getByRole('checkbox', {name: `Mark '${task.title}' as done`})).toBeChecked()

		const stored = await apiContext.get(`tasks/${task.id}`, {headers: {Authorization: `Bearer ${userToken}`}})
		expect(stored.ok()).toBeTruthy()
		expect(await stored.json()).toEqual(expect.objectContaining({
			done: true,
			priority: 3,
		}))
	})

	test('Should not show inline editors in an archived project', async ({authenticatedPage: page}) => {
		await ProjectFactory.create(1, {
			id: 1,
			is_archived: true,
		})
		await createDefaultViews(1)
		const [task] = await TaskFactory.create(1, {
			project_id: 1,
			priority: 3,
		})
		await page.goto('/projects/1/3')
		await showPriorityColumn(page)
		const row = page.locator('.project-table tbody tr').filter({
			has: page.getByRole('link', {name: task.title, exact: true}),
		})

		await expect(row.locator('.priority-label')).toHaveText('High')
		await expect(row.getByRole('checkbox')).toHaveCount(0)
		await expect(row.getByRole('combobox')).toHaveCount(0)
		await expect(row.locator('.datepicker')).toHaveCount(0)
		await expect(row.getByRole('button', {name: 'Assignees'})).toHaveCount(0)
	})

	test('Should navigate to the task when the title is clicked', async ({authenticatedPage: page}) => {
		await createProjects(1)
		await TaskFactory.create(5, {
			id: '{increment}',
			project_id: 1,
		})
		const loadTasksPromise = page.waitForResponse(response =>
			response.url().includes('/projects/1/views/') && response.url().includes('/tasks'),
		)
		await page.goto('/projects/1/3')
		await loadTasksPromise

		await page.locator('.project-table table.table tbody tr').first().locator('a').first().click()

		await expect(page).toHaveURL(/\/tasks\/\d+/)
	})

	test('Should respect filter query parameter from URL', async ({authenticatedPage: page}) => {
		await createProjects(1)
		const {highPriorityTasks, lowPriorityTasks} = await createTasksWithPriorities()

		await page.goto('/projects/1/3?filter=priority%20>=%204')

		await expect(page).toHaveURL(/filter=priority/)

		// Wait for tasks to load and verify high priority tasks are visible
		await expect(page.locator('.project-table table.table')).toContainText(highPriorityTasks[0].title, {timeout: 10000})
		await expect(page.locator('.project-table table.table')).toContainText(highPriorityTasks[1].title)

		// Verify low priority tasks are not visible
		await expect(page.locator('.project-table table.table')).not.toContainText(lowPriorityTasks[0].title)
		await expect(page.locator('.project-table table.table')).not.toContainText(lowPriorityTasks[1].title)
	})

	test('Should respect search query parameter from URL', async ({authenticatedPage: page}) => {
		await createProjects(1)
		const {searchableTask} = await createTasksWithSearch()

		await page.goto('/projects/1/3?s=meeting')

		await expect(page).toHaveURL(/s=meeting/)

		// Wait for search results to load and verify searchable task is visible
		await expect(page.locator('.project-table table.table')).toContainText(searchableTask.title, {timeout: 10000})

		// Verify only one task row is shown (the search result)
		await expect(page.locator('.project-table table.table tbody tr')).toHaveCount(1)
	})

	test('Should focus the picker search and show its list after +', async ({authenticatedPage: page}) => {
		await createProjects(1)
		await TaskFactory.create(1, {project_id: 1})
		const [label] = await LabelFactory.create(1)
		await page.goto('/projects/1/3')

		await page.locator('.project-table tbody tr').first().getByRole('button', {name: 'Labels', exact: true}).click()

		const picker = page.locator('.table-picker')
		await expect(picker.getByRole('combobox')).toBeFocused()
		await expect(picker.getByRole('listbox')).toBeVisible()
		await expect(picker.getByRole('option').filter({hasText: label.title})).toBeVisible()
	})
})
