import {test, expect} from '../../support/fixtures'
import {UserFactory} from '../../factories/user'
import {ProjectFactory} from '../../factories/project'
import {ProjectViewFactory} from '../../factories/project_view'
import {BucketFactory} from '../../factories/bucket'
import {TaskFactory} from '../../factories/task'
import {TaskBucketFactory} from '../../factories/task_buckets'
import {TaskAssigneeFactory} from '../../factories/task_assignee'
import {login} from '../../support/authenticateUser'

const STATUSES = ['Idea', 'To-Do', 'Doing', 'Done']

// Two users with one unshared project each; every project has a status board and one planned, assigned task.
async function seed() {
	const users = await UserFactory.create(2, {
		username: (i: number) => i === 1 ? 'boss' : 'worker',
		is_admin: (i: number) => i === 1,
	})
	await ProjectFactory.create(2, {
		title: (i: number) => i === 1 ? 'Boss project' : 'Worker secret',
		owner_id: (i: number) => i,
	})
	await ProjectViewFactory.create(2, {
		id: (i: number) => i,
		project_id: (i: number) => i,
		title: 'Kanban',
		view_kind: 3,
		bucket_configuration_mode: 1,
		default_bucket_id: (i: number) => i * 4 - 3,
		done_bucket_id: (i: number) => i * 4,
	})
	await BucketFactory.create(8, {
		id: (i: number) => i,
		title: (i: number) => STATUSES[(i - 1) % 4],
		project_view_id: (i: number) => Math.ceil(i / 4),
		position: (i: number) => i,
	})
	await TaskFactory.create(2, {
		id: (i: number) => i,
		index: (i: number) => i,
		project_id: (i: number) => i,
		title: (i: number) => i === 1 ? 'Boss task' : 'Worker task',
		created_by_id: (i: number) => i,
		size: (i: number) => i === 1 ? 4 : 8,
	})
	await TaskBucketFactory.create(2, {
		task_id: (i: number) => i,
		bucket_id: (i: number) => i * 4 - 2,
		project_view_id: (i: number) => i,
	})
	await TaskAssigneeFactory.create(2, {
		task_id: (i: number) => i,
		user_id: (i: number) => i,
	})
	return users
}

test.describe('Admin oversight', () => {
	test('an all-projects filter and the workload page cover unshared projects', async ({page, apiContext}) => {
		const [admin] = await seed()
		const {token} = await login(page, apiContext, admin)
		const headers = {Authorization: `Bearer ${token}`}

		await page.goto('/filters/new')
		await page.locator('input#Title').fill('Everything')
		await page.getByText('All projects', {exact: true}).click()
		await page.locator('button.is-primary.is-fullwidth').click()
		await expect(page).toHaveURL(/\/projects\/-\d+\/[1-9]\d*$/)
		await page.locator('.switch-view-button', {hasText: 'Table'}).click()

		await page.getByRole('button', {name: 'Columns', exact: true}).click()
		await page.locator('.columns-filter').getByText('Project', {exact: true}).click()
		await page.getByRole('button', {name: 'Columns', exact: true}).click()
		const row = page.locator('.project-table tbody tr', {hasText: 'Worker task'})
		await expect(row).toContainText('Worker secret')
		const status = row.locator('.task-status select')
		await expect(status.locator('option:checked')).toHaveText('To-Do')

		await status.selectOption('Doing')
		await expect(status.locator('option:checked')).toHaveText('Doing')
		await expect.poll(async () => {
			const board = await apiContext.get('/api/v2/projects/2/views/2/buckets/tasks', {headers})
			const buckets: {title: string, tasks?: {id: number}[] | null}[] = (await board.json()).items
			return buckets.find(bucket => bucket.tasks?.some(task => task.id === 2))?.title
		}).toBe('Doing')
		await page.reload()
		await expect(row.locator('.task-status select option:checked')).toHaveText('Doing')

		await page.goto('/admin/workload')
		const blocks = page.locator('[data-testid="workload-block"]')
		const workerBlock = blocks.filter({has: page.getByRole('heading', {name: 'worker', exact: true})})
		await expect(blocks.filter({has: page.getByRole('heading', {name: 'boss', exact: true})})).toContainText('4 h')
		await expect(workerBlock).toContainText('8 h')
		await expect(workerBlock).toContainText('Worker secret')
		await expect(page.locator('[data-testid="workload-total"]')).toContainText('12 h')

		await workerBlock.getByRole('link', {name: 'Worker task'}).click()
		const sizeSelect = page.locator('.task-view .columns.details .column').filter({hasText: 'Effort'}).locator('.select select')
		await sizeSelect.selectOption('1 week')
		await expect(page.locator('.global-notification')).toContainText('Success')
		await page.keyboard.press('Escape')
		await expect(page.locator('.task-view')).toHaveCount(0)
		await expect(page).toHaveURL(/\/admin\/workload$/)
		await expect(workerBlock).toContainText('40 h')
		await expect(page.locator('[data-testid="workload-total"]')).toContainText('44 h')
	})

	test('a non-admin gets no all-projects option and no workload page', async ({page, apiContext}) => {
		const [, worker] = await seed()
		await login(page, apiContext, worker)

		await page.goto('/filters/new')
		await expect(page.locator('input#Title')).toBeVisible()
		await expect(page.getByText('All projects', {exact: true})).toHaveCount(0)

		await page.goto('/admin/workload')
		await expect(page).not.toHaveURL(/\/admin\/workload$/)
	})
})
