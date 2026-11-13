import {test, expect} from '../../support/fixtures'
import {openTaskAction} from '../../support/taskDetail'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {UserFactory} from '../../factories/user'
import {UserProjectFactory} from '../../factories/users_project'

// The detail columns used to split the row evenly, so every extra attribute squeezed the
// assignees column (#3709). The properties strip now wraps instead; this guards the same
// symptom: with every property set, the assignee results must stay readable.
test.describe('Assignee search results in a crowded properties strip', () => {
	test('Shows the avatar and name of every result when every property is set', async ({authenticatedPage: page}) => {
		await page.setViewportSize({width: 1280, height: 800})

		// Don't truncate, and start at 100, to keep the fixture's logged-in user (ID 1)
		const users = await UserFactory.create(3, {
			id: (i: number) => 100 + i,
		}, false)
		const projects = await ProjectFactory.create(1)
		const tasks = await TaskFactory.create(1, {
			id: 1,
			project_id: projects[0].id,
			priority: 3,
			percent_done: 0.5,
			size: 4,
			repeat_after: 86_400,
			due_date: new Date(Date.now() + 86_400_000).toISOString(),
			start_date: new Date(Date.now() + 43_200_000).toISOString(),
			end_date: new Date(Date.now() + 172_800_000).toISOString(),
		})
		await UserProjectFactory.create(3, {
			project_id: projects[0].id,
			user_id: (i: number) => users[i - 1].id,
		})

		await page.goto(`/tasks/${tasks[0].id}`)
		await expect(page.locator('.task-view .properties .property')).toHaveCount(7)
		await openTaskAction(page, 'addProperty', 'Assign to User')

		const multiselect = page.locator('.task-view .property.assignees .multiselect')
		await multiselect.locator('input').click()
		await expect(multiselect.locator('.search-results')).toBeVisible({timeout: 5000})

		const results = multiselect.locator('.search-result-button')
		await expect(results.first()).toBeVisible()

		for (const result of await results.all()) {
			// `.user` is what the row clips: when it collapses to zero the row
			// renders blank, avatar and name included.
			const user = await result.locator('.user').boundingBox()
			const avatar = await result.locator('img.avatar').boundingBox()

			expect(avatar?.width).toBeGreaterThan(0)
			expect(user?.width).toBeGreaterThan(avatar?.width ?? 0)
		}
	})
})
