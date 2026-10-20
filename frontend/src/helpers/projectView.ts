import type {ProjectView} from '@/client/generated'
import {PROJECT_VIEW_KINDS} from '@/constants/projectView'

export type ProjectViewSettings = Record<number, number>

const SETTINGS_KEY_PROJECT_VIEW = 'projectView'

/**
 * Save the current project view to local storage
 */
export function saveProjectView(projectId: number, viewId: number) {
	if (!projectId || !viewId) {
		return
	}

	// We use local storage and not the store here to make it persistent across reloads.
	const savedProjectView = localStorage.getItem(SETTINGS_KEY_PROJECT_VIEW)
	let savedProjectViewSettings: ProjectViewSettings | false = false
	if (savedProjectView !== null) {
		savedProjectViewSettings = JSON.parse(savedProjectView) as ProjectViewSettings
	}

	let projectViewSettings: ProjectViewSettings = {}
	if (savedProjectViewSettings) {
		projectViewSettings = savedProjectViewSettings
	}

	projectViewSettings[projectId] = viewId
	localStorage.setItem(SETTINGS_KEY_PROJECT_VIEW, JSON.stringify(projectViewSettings))
}

export function getProjectViewId(projectId: number): number {
	const projectViewSettingsString = localStorage.getItem(SETTINGS_KEY_PROJECT_VIEW)
	if (!projectViewSettingsString) {
		return 0
	}

	const projectViewSettings = JSON.parse(projectViewSettingsString) as ProjectViewSettings
	if (isNaN(projectViewSettings[projectId])) {
		return 0
	}
	return projectViewSettings[projectId]
}

// A task's status is its bucket in the project's only manual kanban view, or the active one when there are several.
export function getStatusKanbanView(views: ProjectView[] | undefined, activeViewId: number | undefined) {
	const manual = (views ?? []).filter(view => view.view_kind === PROJECT_VIEW_KINDS.KANBAN
		&& view.bucket_configuration_mode === 'manual')
	return manual.length === 1
		? manual[0]
		: manual.find(view => view.id === activeViewId) ?? null
}
