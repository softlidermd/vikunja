import {test, expect} from '../../support/fixtures'
import {TaskFactory} from '../../factories/task'
import {createProjects} from './prepareProjects'

test.describe('Subtasks in the table view', () => {
	test('nests a subtask under its parent and collapses it', async ({authenticatedPage: page, apiContext, userToken}) => {
		await createProjects(1)
		const [parent, child] = await TaskFactory.create(2, {
			project_id: 1,
			title: (i: number) => i === 1 ? 'Parent task' : 'Child task',
		})
		const relation = await apiContext.post(`/api/v2/tasks/${parent.id}/relations`, {
			headers: {Authorization: `Bearer ${userToken}`},
			data: {other_task_id: child.id, relation_kind: 'subtask'},
		})
		expect(relation.ok(), await relation.text()).toBeTruthy()

		await page.goto('/projects/1/3')

		// The default sort is newest first, so without nesting the child would come first.
		const rows = page.locator('.project-table tbody tr')
		await expect(rows.nth(0)).toContainText('Parent task')
		await expect(rows.nth(1)).toContainText('Child task')

		await page.getByRole('button', {name: 'Hide subtasks of Parent task'}).click()
		await expect(page.getByRole('link', {name: 'Child task', exact: true})).toHaveCount(0)
		await page.getByRole('button', {name: 'Show subtasks of Parent task'}).click()
		await expect(page.getByRole('link', {name: 'Child task', exact: true})).toBeVisible()
	})

	test('adds a subtask from the row menu', async ({authenticatedPage: page, apiContext, userToken}) => {
		await createProjects(1)
		const [parent] = await TaskFactory.create(1, {project_id: 1, title: 'Parent task'})

		await page.goto('/projects/1/3')
		await page.getByRole('button', {name: 'Actions for Parent task'}).click()
		await page.getByRole('button', {name: 'Add subtask'}).click()
		// Typed without clicking: the title field must have the focus when the dialog opens.
		await expect(page.getByPlaceholder('Add a subtask…')).toBeFocused()
		await page.keyboard.type('New child')
		await page.keyboard.press('Enter')

		const rows = page.locator('.project-table tbody tr')
		await expect(rows.nth(1)).toContainText('New child')

		const response = await apiContext.get(`/api/v2/tasks/${parent.id}`, {
			headers: {Authorization: `Bearer ${userToken}`},
		})
		const body = await response.json()
		expect(body.related_tasks.subtask.map((t: {title: string}) => t.title)).toEqual(['New child'])
	})

	test('adds a subtask from the task detail of a task without relations', async ({authenticatedPage: page}) => {
		await createProjects(1)
		const [parent] = await TaskFactory.create(1, {project_id: 1, title: 'Parent task'})

		await page.goto(`/tasks/${parent.id}`)
		await page.locator('.task-view .task-subtasks__add').click()
		// The field opens focused and stays open for the next subtask.
		await page.keyboard.type('Detail child')
		await page.keyboard.press('Enter')

		await expect(page.locator('.task-subtasks__item').filter({hasText: 'Detail child'})).toBeVisible()
		await expect(page.getByText('Subtasks · 0 of 1')).toBeVisible()
		await expect(page.getByPlaceholder('Add a subtask…')).toHaveValue('')
		await expect(page.getByPlaceholder('Add a subtask…')).toBeFocused()
	})
})
