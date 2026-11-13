import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {createDefaultViews} from '../project/prepareProjects'

test('task subscription survives reload and can be removed', async ({authenticatedPage: page, apiContext, userToken}) => {
	const [project] = await ProjectFactory.create(1)
	await createDefaultViews(project.id)
	const [task] = await TaskFactory.create(1, {project_id: project.id})
	const headers = {Authorization: `Bearer ${userToken}`}
	await page.goto(`/tasks/${task.id}`)
	const more = page.locator('.task-view [data-cy="taskDetail.more"]')
	// Each item closes the menu, so reopen it before every assertion.
	const menuItem = async (name: string) => {
		await more.locator('.dropdown-trigger').click()
		return more.getByRole('button', {name, exact: true})
	}
	await (await menuItem('Subscribe')).click()
	await expect(await menuItem('Unsubscribe')).toBeVisible()
	await page.reload()
	await (await menuItem('Unsubscribe')).click()
	await expect(await menuItem('Subscribe')).toBeVisible()
	await page.reload()
	await expect(await menuItem('Subscribe')).toBeVisible()
	const stored = await apiContext.get(`tasks/${task.id}`, {headers})
	expect(stored.ok()).toBeTruthy()
	expect((await stored.json()).subscription).toBeFalsy()
})
