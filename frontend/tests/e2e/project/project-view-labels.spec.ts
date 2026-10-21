import type {APIRequestContext, Page} from '@playwright/test'
import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {ProjectViewFactory} from '../../factories/project_view'
import {TaskFactory} from '../../factories/task'
import {LabelFactory} from '../../factories/labels'
import {LabelTaskFactory} from '../../factories/label_task'

const VIEW_ID = 5

// Labels "Alpha", "beta" and "Gamma" (ids 1-3) and a labels view (kind 100) on project 1.
async function seedBoard(filter?: string) {
	await ProjectFactory.create(1)
	await ProjectViewFactory.create(1, {
		id: VIEW_ID,
		project_id: 1,
		view_kind: 100,
		...(filter ? {filter: JSON.stringify({filter, filter_include_nulls: false, s: ''})} : {}),
	})
	const titles = ['Gamma', 'Alpha', 'beta']
	await LabelFactory.create(3, {
		id: '{increment}',
		title: (i: number) => titles[i - 1],
	})
}

async function seedTasks(labelsPerTask: number[][]) {
	const tasks = await TaskFactory.create(labelsPerTask.length, {
		id: '{increment}',
		index: '{increment}',
		title: (i: number) => `Task ${i}`,
	})
	const pairs = labelsPerTask.flatMap((labels, index) => labels.map(label => ({task: index + 1, label})))
	for (const [index, {task, label}] of pairs.entries()) {
		await LabelTaskFactory.create(1, {id: index + 1, task_id: task, label_id: label}, index === 0)
	}
	return tasks
}

const card = (page: Page, title: string) => page.locator('.label-card').filter({
	has: page.locator('.label-card__title', {hasText: new RegExp(`^${title}$`)}),
})

const drag = (page: Page, from: string, task: string, to: string) => card(page, from)
	.locator('.single-task')
	.filter({hasText: task})
	.dragTo(card(page, to).locator('.label-card__tasks'))

async function getTask(apiContext: APIRequestContext, token: string, taskId: number) {
	const response = await apiContext.get(`tasks/${taskId}`, {headers: {Authorization: `Bearer ${token}`}})
	expect(response.ok()).toBe(true)
	return response.json()
}

async function taskLabelIds(apiContext: APIRequestContext, token: string, taskId: number) {
	return ((await getTask(apiContext, token, taskId)).labels ?? []).map((label: {id: number}) => label.id).sort()
}

test.describe('Project View Labels', () => {
	test('Shows one card per label in alphabetical order, the no-label card last', async ({
		authenticatedPage: page,
	}) => {
		await seedBoard()
		// Task 1 has Alpha and beta, task 2 has no label. Gamma is on no task, so it gets no card.
		await seedTasks([[2, 3], []])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await expect(page.locator('.label-card__title')).toHaveText(['Alpha', 'beta', 'No label'])
		await expect(card(page, 'Alpha')).toContainText('Task 1')
		await expect(card(page, 'beta')).toContainText('Task 1')
		await expect(card(page, 'No label')).toContainText('Task 2')
		await expect(card(page, 'Alpha').locator('.label-card__count')).toHaveText('1')
	})

	test('Adds a task with the card label', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard()
		await seedTasks([[2]])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await card(page, 'Alpha').getByRole('button', {name: 'Add a task'}).click()
		const input = card(page, 'Alpha').locator('.label-card__footer input')
		await expect(input).toBeFocused()
		await input.fill('Card task')
		await input.press('Enter')

		const row = card(page, 'Alpha').locator('[data-task-id]').filter({hasText: 'Card task'})
		await expect(row).toBeVisible()
		await expect(input).toHaveValue('')
		const taskId = Number(await row.getAttribute('data-task-id'))
		await expect.poll(() => taskLabelIds(apiContext, userToken, taskId)).toEqual([2])
	})

	test('A label that fails to attach keeps the task once and clears the input', async ({
		authenticatedPage: page,
	}) => {
		await seedBoard()
		await seedTasks([[2]])
		await page.route('**/tasks/*/labels', route => route.fulfill({
			status: 500,
			json: {message: 'Internal error'},
		}))
		await page.goto(`/projects/1/${VIEW_ID}`)

		await card(page, 'Alpha').getByRole('button', {name: 'Add a task'}).click()
		const input = card(page, 'Alpha').locator('.label-card__footer input')
		await input.fill('Card task')
		await input.press('Enter')

		await expect(page.locator('.global-notification')).toContainText('the label “Alpha” could not be added')
		await expect(input).toHaveValue('')
		await page.reload()
		await expect(card(page, 'No label').locator('.single-task').filter({hasText: 'Card task'})).toHaveCount(1)
	})

	test('Dragging from one card to another relabels the task', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard()
		await seedTasks([[2], [3], [2]])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await drag(page, 'Alpha', 'Task 1', 'beta')
		await expect.poll(() => taskLabelIds(apiContext, userToken, 1)).toEqual([3])

		await page.reload()
		await expect(card(page, 'beta')).toContainText('Task 1')
		await expect(card(page, 'Alpha')).not.toContainText('Task 1')
	})

	test('Dragging between a label and the no-label card adds or removes only that label', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard()
		await seedTasks([[2], [], [2]])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await drag(page, 'No label', 'Task 2', 'Alpha')
		await expect.poll(() => taskLabelIds(apiContext, userToken, 2)).toEqual([2])

		await drag(page, 'Alpha', 'Task 1', 'No label')
		await expect.poll(() => taskLabelIds(apiContext, userToken, 1)).toEqual([])
		await expect(card(page, 'No label')).toContainText('Task 1')
	})

	test('Dragging A to B when the task already has B and C only removes A', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard()
		// Task 1 has Gamma, Alpha and beta; task 2 keeps the beta card alive.
		await seedTasks([[1, 2, 3], [3]])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await drag(page, 'Alpha', 'Task 1', 'beta')

		await expect.poll(() => taskLabelIds(apiContext, userToken, 1)).toEqual([1, 3])
		await expect(page.locator('.label-card__title')).toHaveText(['beta', 'Gamma', 'No label'])
		await expect(card(page, 'beta').locator('.single-task').filter({hasText: 'Task 1'})).toHaveCount(1)
		await expect(card(page, 'Gamma')).toContainText('Task 1')
	})

	test('Ticking a task done removes it from every card', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard()
		await seedTasks([[2, 3], [2]])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await card(page, 'Alpha').locator('.single-task').filter({hasText: 'Task 1'}).locator('.fancy-checkbox').click()

		await expect(page.locator('.label-card .single-task').filter({hasText: 'Task 1'})).toHaveCount(0)
		await expect(card(page, 'Alpha')).toContainText('Task 2')
		await expect.poll(async () => (await getTask(apiContext, userToken, 1)).done).toBe(true)
		await page.reload()
		await expect(card(page, 'Alpha')).toContainText('Task 2')
		await expect(page.locator('.label-card .single-task').filter({hasText: 'Task 1'})).toHaveCount(0)
	})

	test('A label-filtered view shows tasks added to the card and drops tasks dragged out', async ({
		authenticatedPage: page,
		apiContext,
		userToken,
	}) => {
		await seedBoard('labels in 2')
		await seedTasks([[2], []])
		await page.goto(`/projects/1/${VIEW_ID}`)

		await expect(card(page, 'Alpha')).toContainText('Task 1')
		await expect(page.locator('.label-card .single-task').filter({hasText: 'Task 2'})).toHaveCount(0)

		await card(page, 'Alpha').getByRole('button', {name: 'Add a task'}).click()
		const input = card(page, 'Alpha').locator('.label-card__footer input')
		await input.fill('Filtered task')
		await input.press('Enter')
		await expect(card(page, 'Alpha')).toContainText('Filtered task')

		await drag(page, 'Alpha', 'Task 1', 'No label')
		await expect(page.locator('.label-card .single-task').filter({hasText: 'Task 1'})).toHaveCount(0)
		await expect.poll(() => taskLabelIds(apiContext, userToken, 1)).toEqual([])

		await page.reload()
		await expect(card(page, 'Alpha')).toContainText('Filtered task')
		await expect(page.locator('.label-card .single-task').filter({hasText: 'Task 1'})).toHaveCount(0)
	})
})
