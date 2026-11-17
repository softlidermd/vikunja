interface RelatedRef {id?: number}

export interface NestableTask {
	id: number,
	related_tasks?: {subtask?: RelatedRef[] | null, parenttask?: RelatedRef[] | null} | null,
}

export interface NestedTaskRow<T> {
	task: T,
	depth: number,
	hasChildren: boolean,
}

/**
 * Orders a page of tasks so each subtask follows its parent, one level deeper.
 * Only relations inside the page nest; a subtask whose parent is not loaded stays a top-level row.
 * Siblings keep their order in `tasks` (the server sort). A subtask with several parents on the page
 * appears once, under the first one reached. Descendants of a collapsed task are left out.
 */
export function nestTaskRows<T extends NestableTask>(
	tasks: T[],
	collapsed: ReadonlySet<number> = new Set(),
): NestedTaskRow<T>[] {
	const onPage = new Set(tasks.map(t => t.id))
	const childIds = new Map(tasks.map(t => [t.id, new Set((t.related_tasks?.subtask ?? []).map(s => s.id))]))
	const childrenOf = (parent: T) => tasks.filter(t => t.id !== parent.id
		&& childIds.get(parent.id)!.has(t.id))
	const isRoot = (t: T) => !(t.related_tasks?.parenttask ?? [])
		.some(p => p.id !== undefined && p.id !== t.id && onPage.has(p.id))

	const rows: NestedTaskRow<T>[] = []
	const visited = new Set<number>()

	function walk(task: T, depth: number, hidden: boolean) {
		visited.add(task.id)
		const children = childrenOf(task).filter(c => !visited.has(c.id))
		if (!hidden) rows.push({task, depth, hasChildren: children.length > 0})
		// Mark the children before descending so a sibling's subtree cannot claim them first.
		children.forEach(c => visited.add(c.id))
		const hideChildren = hidden || collapsed.has(task.id)
		children.forEach(c => walk(c, depth + 1, hideChildren))
	}

	tasks.filter(isRoot).forEach(t => {
		if (!visited.has(t.id)) walk(t, 0, false)
	})
	// A relation cycle has no root on the page: show what is left flat rather than dropping it.
	tasks.forEach(t => {
		if (!visited.has(t.id)) walk(t, 0, false)
	})
	return rows
}
