<template>
	<!-- A DOM child of the row: keep its clicks, keys and presses from opening the task or starting a drag. -->
	<span
		class="task-quick-actions"
		@click.stop
		@keyup.stop
		@pointerdown.stop
		@mousedown.stop
		@touchstart.stop
	>
		<Popup
			v-model:open="open"
			placement="bottom-end"
			:anchor="triggerEl"
			sheet-on-mobile
			:sheet-title="task.title"
		>
			<template #trigger="{toggle}">
				<BaseButton
					ref="trigger"
					class="task-quick-actions__trigger"
					:aria-label="$t('task.quickActions.label', {task: task.title})"
					:aria-expanded="open ? 'true' : 'false'"
					@click="toggle()"
					@pointerenter="wantsBuckets = true"
					@focus="wantsBuckets = true"
				>
					<Icon icon="ellipsis-v" />
				</BaseButton>
			</template>
			<template #content="{isOpen, close}">
				<div
					v-if="isOpen"
					ref="menu"
					class="task-quick-actions__menu"
					role="group"
					:aria-label="$t('task.quickActions.label', {task: task.title})"
				>
					<template v-if="canWrite && kanbanView && buckets.length > 0">
						<p class="task-quick-actions__heading">
							{{ $t('task.attributes.status') }}
						</p>
						<DropdownItem
							v-for="bucket in buckets"
							:key="bucket.id"
							icon="check"
							:icon-class="{'is-invisible': bucket.id !== currentBucket?.id}"
							:aria-pressed="bucket.id === currentBucket?.id ? 'true' : 'false'"
							@click="run(close, () => moveToBucket(bucket.id!))"
						>
							{{ bucket.title }}
						</DropdownItem>
						<hr class="task-quick-actions__divider">
					</template>
					<DropdownItem
						v-if="canAssign"
						icon="user"
						@click="run(close, toggleAssignMe)"
					>
						{{ isAssignedToMe ? $t('task.quickActions.unassignMe') : $t('task.quickActions.assignMe') }}
					</DropdownItem>
					<DropdownItem
						icon="link"
						@click="run(close, copyLink)"
					>
						{{ $t('admin.inviteLinks.copy') }}
					</DropdownItem>
					<DropdownItem
						v-if="canWrite"
						icon="trash-alt"
						class="has-text-danger"
						@click="close(); showDeleteModal = true"
					>
						{{ $t('task.detail.actions.delete') }}
					</DropdownItem>
				</div>
			</template>
		</Popup>

		<Modal
			:enabled="showDeleteModal"
			@close="showDeleteModal = false"
			@submit="deleteTask()"
		>
			<template #header>
				<span>{{ $t('task.detail.delete.header') }}</span>
			</template>

			<template #text>
				<p class="tw:text-balance">
					{{ $t('task.detail.delete.text1') }}
				</p>
				<p class="tw:text-balance">
					{{ $t('task.detail.delete.text2') }}
				</p>
			</template>
		</Modal>
	</span>
</template>

<script setup lang="ts">
import {computed, nextTick, ref, watch} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import type {Task} from '@/client/generated'
import {
	useAddTaskAssigneeMutation,
	useDeleteTaskMutation,
	useRemoveTaskAssigneeMutation,
} from '@/client/queries/taskMutations'
import BaseButton from '@/components/base/BaseButton.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import Icon from '@/components/misc/Icon'
import Modal from '@/components/misc/Modal.vue'
import Popup from '@/components/misc/Popup.vue'
import {useCopyToClipboard} from '@/composables/useCopyToClipboard'
import {useTaskSaveQueue} from '@/composables/useTaskSaveQueue'
import {useTaskStatus} from '@/composables/useTaskStatus'
import {success} from '@/message'
import {useAuthStore} from '@/stores/auth'

const props = defineProps<{
	task: Task,
	canWrite: boolean,
}>()

const {t} = useI18n({useScope: 'global'})
const router = useRouter()
const authStore = useAuthStore()
const copy = useCopyToClipboard()
const addAssignee = useAddTaskAssigneeMutation()
const removeAssignee = useRemoveTaskAssigneeMutation()
const deleteMutation = useDeleteTaskMutation()
const {queueTaskSave} = useTaskSaveQueue()

const open = ref(false)
const showDeleteModal = ref(false)
// Loaded when the trigger is pointed at, not per row up front, and before the menu opens so it does not grow under the cursor.
const wantsBuckets = ref(false)

const {kanbanView, buckets, currentBucket, moveToBucket} = useTaskStatus(
	() => props.task,
	() => (wantsBuckets.value || open.value) && props.canWrite,
)

const trigger = ref<InstanceType<typeof BaseButton> | null>(null)
const triggerEl = computed<HTMLElement | null>(() => trigger.value?.$el ?? null)
const menu = ref<HTMLElement | null>(null)

watch(menu, el => el?.querySelector<HTMLElement>('button')?.focus())

// Escape or an action hides the menu with the focus inside it; give the focus back to the trigger.
watch(open, isOpen => {
	const active = document.activeElement
	if (!isOpen && (active === document.body || menu.value?.contains(active))) {
		triggerEl.value?.focus()
	}
}, {flush: 'pre'})

const me = computed(() => authStore.info)
const canAssign = computed(() => props.canWrite && !authStore.isLinkShareAuth && me.value?.id !== undefined)
const isAssignedToMe = computed(() => (props.task.assignees ?? []).some(user => user.id === me.value?.id))

async function run(close: () => void, action: () => Promise<unknown>) {
	close()
	try {
		await action()
	} catch {
		// Failed mutations already show an error toast (contextMutationOptions).
	}
}

async function toggleAssignMe() {
	const user = me.value
	if (user?.id === undefined || props.task.id === undefined) {
		return
	}
	const input = {
		taskId: props.task.id,
		user: {...user, id: user.id},
	}
	const remove = isAssignedToMe.value
	// Queued: a status move still in flight answers with the whole task and would drop the new assignee.
	await queueTaskSave(input.taskId, async () => {
		await (remove ? removeAssignee.mutateAsync(input) : addAssignee.mutateAsync(input))
	})
}

async function copyLink() {
	const {href} = router.resolve({name: 'task.detail', params: {id: props.task.id}})
	if (await copy(new URL(href, window.location.href).href)) {
		success({message: t('admin.inviteLinks.copied')})
	}
}

async function deleteTask() {
	if (deleteMutation.isPending.value) {
		return
	}
	// The row, its trigger and this dialog go away with the task, so the focus moves to a neighbouring row.
	const row = triggerEl.value?.closest('[data-task-id], tr')
	const neighbour = row?.nextElementSibling ?? row?.previousElementSibling
	await deleteMutation.mutateAsync(props.task.id!)
	showDeleteModal.value = false
	success({message: t('task.detail.deleteSuccess')})
	await nextTick()
	neighbour?.querySelector<HTMLElement>('.task-quick-actions__trigger, a[href]')?.focus()
}
</script>

<style lang="scss" scoped>
.task-quick-actions {
	display: inline-flex;
	flex-shrink: 0;
}

.task-quick-actions__trigger {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	inline-size: 1.75rem;
	block-size: 1.75rem;
	border-radius: $radius;
	color: var(--grey-500);
	transition: opacity $transition, background-color $transition;

	&:hover,
	&[aria-expanded='true'] {
		background-color: var(--grey-200);
		color: var(--text);
	}
}

// Shown on the row's hover like the List row's star; touch has no hover, so it stays visible there.
// :focus, not :focus-visible: the focus returned after Escape or an action is programmatic.
@media (hover: hover) and (pointer: fine) {
	.task-quick-actions__trigger:not(:focus, [aria-expanded='true']) {
		opacity: 0;
	}

	:is(.board-task, .single-task, tr):hover .task-quick-actions__trigger {
		opacity: 1;
	}
}

.task-quick-actions__menu {
	min-inline-size: 12rem;
	padding-block: .5rem;
	border-radius: $radius;
	background-color: var(--scheme-main);
	box-shadow: var(--shadow-md);
}

.task-quick-actions__heading {
	margin: 0;
	padding: .25rem 1rem;
	color: var(--grey-500);
	font-size: .75rem;
	text-transform: uppercase;
}

.task-quick-actions__divider {
	block-size: 1px;
	margin: .5rem 0;
	border: none;
	background-color: var(--border-light);
}
</style>
