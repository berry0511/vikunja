<template>
	<div
		:data-task-id="task.id"
		:data-project-id="task.project_id"
	>
		<div
			ref="taskRoot"
			:class="{'is-loading': isLoading}"
			class="task loader-container single-task"
			tabindex="-1"
			:data-is-overdue="isOverdue || undefined"
			@click="openTaskDetail"
			@keyup.enter="openTaskDetail"
		>
			<span
				v-tooltip="!canMarkAsDone ? $t('task.readOnlyCheckbox') : ''"
				class="is-inline-flex is-align-items-center"
			>
				<FancyCheckbox
					:model-value="task.done ?? false"
					:disabled="isArchived || disabled || !canMarkAsDone"
					:aria-label="$t('task.detail.markAsDone', {task: task.title})"
					@update:modelValue="markAsDone"
					@click.stop
				/>
			</span>

			<ColorBubble
				v-if="!showProjectSeparately && projectColor !== '' && currentProject?.id !== task.project_id"
				:color="projectColor"
				class="mie-1"
			/>

			<div
				:class="{ 'done': task.done, 'show-project': showProject && project}"
				class="tasktext"
			>
				<span>
					<RouterLink
						v-if="showProject && typeof project !== 'undefined'"
						v-tooltip="$t('task.detail.belongsToProject', {project: project.title})"
						:to="{ name: 'project.index', params: { projectId: task.project_id } }"
						class="task-project mie-1"
						:class="{'mie-2': task.hex_color !== ''}"
						@click.stop
					>
						{{ project.title }}
					</RouterLink>

					<ColorBubble
						v-if="task.hex_color !== ''"
						:color="getHexColor(task.hex_color)"
						class="mie-1"
					/>
	
					<PriorityLabel
						:priority="task.priority ?? 0"
						:done="task.done"
						class="pis-2 mie-1"
					/>

					<TaskGlanceTooltip :task="task">
						<RouterLink
							ref="taskLinkRef"
							:to="taskDetailRoute"
							class="task-link"
						>
							{{ task.title }}
						</RouterLink>
					</TaskGlanceTooltip>
				</span>

				<Labels
					v-if="(task.labels?.length ?? 0) > 0"
					class="labels mis-2 mie-1"
					:labels="task.labels ?? []"
				/>

				<AssigneeList
					v-if="(task.assignees?.length ?? 0) > 0"
					:assignees="task.assignees ?? []"
					:avatar-size="25"
					class="mis-1"
					:inline="true"
				/>

				<Popup
					v-if="task.due_date && +new Date(task.due_date) > 0"
					placement="bottom-start"
					:anchor="dueDateTriggerEl"
					sheet-on-mobile
					:sheet-title="$t('task.deferDueDate.title')"
				>
					<template #trigger="{toggle, isOpen}">
						<BaseButton
							ref="dueDateTrigger"
							v-tooltip="formatDateLong(task.due_date)"
							class="dueDate"
							@click.prevent.stop="toggle()"
						>	
							<time
								:datetime="formatISO(task.due_date)"
								class="is-italic"
								:aria-expanded="isOpen ? 'true' : 'false'"
							>
								– {{ $t('task.detail.due', {at: dueDateFormatted}) }}
							</time>
						</BaseButton>
					</template>
					<template #content="{isOpen}">
						<DeferTask
							v-if="isOpen"
							:model-value="task"
						/>
					</template>
				</Popup>

				<span>
					<span
						v-if="(task.attachments?.length ?? 0) > 0"
						class="project-task-icon"
						role="img"
						:aria-label="$t('task.attributes.attachment', (task.attachments?.length ?? 0))"
					>
						<Icon icon="paperclip" />
					</span>
					<span
						v-if="!isEditorContentEmpty((task.description ?? ''))"
						class="project-task-icon is-mirrored-rtl"
					>
						<Icon icon="align-left" />
					</span>
					<span
						v-if="isRepeating"
						class="project-task-icon"
					>
						<Icon icon="history" />
					</span>
					<CommentCount
						:task="task"
						class="project-task-icon"
					/>
				</span>

				<ChecklistSummary :task="task" />
			</div>

			<ProgressBar
				v-if="(task.percent_done ?? 0) > 0"
				:value="(task.percent_done ?? 0) * 100"
				is-small
			/>

			<ColorBubble
				v-if="showProjectSeparately && projectColor !== '' && currentProject?.id !== task.project_id"
				:color="projectColor"
				class="mie-1"
			/>

			<RouterLink
				v-if="showProjectSeparately"
				v-tooltip="$t('task.detail.belongsToProject', {project: project.title})"
				:to="{ name: 'project.index', params: { projectId: task.project_id } }"
				class="task-project"
				@click.stop
			>
				{{ project.title }}
			</RouterLink>

			<BaseButton
				:class="{'is-favorite': task.is_favorite}"
				class="favorite"
				@click.stop="toggleFavorite"
			>
				<span class="is-sr-only">{{
					task.is_favorite ? $t('task.detail.actions.unfavorite') : $t('task.detail.actions.favorite')
				}}</span>
				<Icon
					v-if="task.is_favorite"
					icon="star"
				/>
				<Icon
					v-else
					:icon="['far', 'star']"
				/>
			</BaseButton>
			<slot />
		</div>
		<template v-if="typeof task.related_tasks?.subtask !== 'undefined'">
			<template v-for="subtask in task.related_tasks.subtask">
				<template v-if="getTaskById(subtask.id)">
					<single-task-in-project
						:key="subtask.id"
						:the-task="getTaskById(subtask.id)!"
						:disabled="disabled"
						:can-mark-as-done="canMarkAsDone"
						:all-tasks="allTasks"
						class="subtask-nested"
					/>
				</template>
			</template>
		</template>
	</div>
</template>

<script setup lang="ts">
import {ref, watch, onMounted, computed} from 'vue'
import {useI18n} from 'vue-i18n'

import {getHexColor} from '@/helpers/task'
import type {Task as ITask} from '@/client/generated'

import PriorityLabel from '@/components/tasks/partials/PriorityLabel.vue'
import Labels from '@/components/tasks/partials/Labels.vue'
import TaskGlanceTooltip from '@/components/tasks/partials/TaskGlanceTooltip.vue'
import DeferTask from '@/components/tasks/partials/DeferTask.vue'
import ChecklistSummary from '@/components/tasks/partials/ChecklistSummary.vue'
import CommentCount from '@/components/tasks/partials/CommentCount.vue'

import ProgressBar from '@/components/misc/ProgressBar.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import ColorBubble from '@/components/misc/ColorBubble.vue'
import Popup from '@/components/misc/Popup.vue'


import {formatDisplayDate, formatISO, formatDateLong} from '@/helpers/time/formatDate'
import {success} from '@/message'

import {useProjects} from '@/composables/useProjects'
import {useCurrentProject} from '@/composables/useCurrentProject'
import {useUpdateTaskMutation, useFavoriteTaskMutation} from '@/client/queries/taskMutations'
import AssigneeList from '@/components/tasks/partials/AssigneeList.vue'
import {useIntervalFn} from '@vueuse/core'
import {playPopSound} from '@/helpers/playPop'
import {isEditorContentEmpty} from '@/helpers/editorContentEmpty'
import {TASK_REPEAT_MODES} from '@/types/IRepeatMode'
import {useGlobalNow} from '@/composables/useGlobalNow'
import {useDelayedLoading} from '@/composables/useDelayedLoading'

const props = withDefaults(defineProps<{
	theTask: ITask,
	isArchived?: boolean,
	showProject?: boolean,
	disabled?: boolean,
	canMarkAsDone?: boolean,
	allTasks?: ITask[],
}>(), {
	isArchived: false,
	showProject: false,
	disabled: false,
	canMarkAsDone: true,
	allTasks: () => [],
})

const emit = defineEmits<{
	'taskUpdated': [task: ITask],
}>()

function getTaskById(taskId: number | undefined): ITask | undefined {
	if (typeof props.allTasks === 'undefined' || props.allTasks.length === 0) {
		return undefined
	}

	return props.allTasks.find(t => t.id === taskId)
}

const {t} = useI18n({useScope: 'global'})

const task = computed(() => props.theTask)

const isRepeating = computed(() => (task.value.repeat_after ?? 0) > 0
	|| ((task.value.repeat_after ?? 0) === 0
		&& task.value.repeat_mode === TASK_REPEAT_MODES.REPEAT_MODE_MONTH))

const projectList = useProjects()
const updateTask = useUpdateTaskMutation(true)
const favoriteTask = useFavoriteTaskMutation()
const isLoading = useDelayedLoading(() => updateTask.isPending.value || favoriteTask.isPending.value)

const project = computed(() => projectList.projects[task.value.project_id ?? 0])
const projectColor = computed(() => project.value?.hex_color ?? '')

const showProjectSeparately = computed(() => !props.showProject
	&& currentProject.value?.id !== task.value.project_id
	&& project.value)

const {currentProject} = useCurrentProject()

const taskDetailRoute = computed(() => ({
	name: 'task.detail',
	params: {id: task.value.id},
	// TODO: re-enable opening task detail in modal
	// state: { backdropView: router.currentRoute.value.fullPath },
}))

function updateDueDate() {
	if (!task.value.due_date) {
		return
	}

	dueDateFormatted.value = formatDisplayDate(task.value.due_date)
}

const dueDateFormatted = ref('')
useIntervalFn(updateDueDate, 60_000, {
	immediateCallback: true,
})
onMounted(updateDueDate)

watch(() => task.value.due_date, updateDueDate)

const {now} = useGlobalNow()
const isOverdue = computed(() => (
	!task.value.done &&
	task.value.due_date !== null &&
	new Date(task.value.due_date ?? 0).getTime() > 0 &&
	new Date(task.value.due_date ?? 0).getTime() <= now.value.getTime()
))

let oldTask: ITask

async function markAsDone(checked: boolean, wasReverted: boolean = false) {
	if (!wasReverted) oldTask = {...task.value}

	// Fire the request immediately and with the intended done value snapshotted, so a re-render or
	// teardown during the animation delay can neither drop the save nor make it send a stale state.
	const source = wasReverted && isRepeating.value ? oldTask : task.value
	const updatePromise = updateTask.mutateAsync({
		...source,
		id: source.id!,
		done: checked,
	}).catch(() => undefined)

	const finish = async () => {
		const newTask = await updatePromise
		if (!newTask) return

		updateDueDate()

		if (wasReverted) {
			return
		}

		if (checked) {
			playPopSound()
		}
		emit('taskUpdated', newTask)

		let message = t('task.doneSuccess')
		if (!task.value.done && !isRepeating.value) {
			message = t('task.undoneSuccess')
		}

		success({message}, [{
			title: t('task.undo'),
			callback: () => undoDone(checked),
		}])
	}

	if (checked) {
		setTimeout(finish, 300) // Delay only the follow-up to show the animation when marking a task as done
	} else {
		await finish() // Don't delay it when un-marking it as it doesn't have an animation the other way around
	}
}

function undoDone(checked: boolean) {
	markAsDone(!checked, true)
}

async function toggleFavorite() {
	const updated = await favoriteTask.mutateAsync({...task.value, id: task.value.id!})
	emit('taskUpdated', updated)
}

const taskRoot = ref<HTMLElement | null>(null)
const dueDateTrigger = ref<InstanceType<typeof BaseButton> | null>(null)
const dueDateTriggerEl = computed<HTMLElement | null>(() => dueDateTrigger.value?.$el ?? null)
const taskLinkRef = ref<InstanceType<typeof BaseButton> | null>(null)

function hasTextSelected() {
	const isTextSelected = window.getSelection()?.toString()
	return !(typeof isTextSelected === 'undefined' || isTextSelected === '' || isTextSelected === '\n')
}

function openTaskDetail(event: MouseEvent | KeyboardEvent) {
	if (event.target instanceof HTMLElement) {
		const isInteractiveElement = event.target.closest('a, button, label, input[type="checkbox"], .favorite, [role="button"]')
		if (isInteractiveElement || hasTextSelected()) {
			return
		}
	}

	taskLinkRef.value?.$el.click()
}

defineExpose({
	focus: () => taskRoot.value?.focus(),
	click: (e: MouseEvent | KeyboardEvent) => openTaskDetail(e),
})
</script>

<style lang="scss" scoped>
/* Microsoft To Do-inspired flat, airy row shared by project and overview lists. */
.task {
	min-block-size: 3.35rem;
	padding: .65rem .9rem;
	margin: .2rem 0;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 9px;
	box-shadow: 0 1px 2px rgba(15, 23, 42, .025);
	gap: .45rem;
	transition: background-color .12s ease, border-color .12s ease, box-shadow .12s ease;
}
.task:hover {
	background: var(--grey-50);
	border-color: color-mix(in srgb, var(--primary) 32%, var(--card-border-color));
	box-shadow: 0 2px 8px rgba(15, 23, 42, .045);
}
.task:focus-within {
	outline: 2px solid color-mix(in srgb, var(--primary) 28%, transparent);
	outline-offset: 1px;
}
.task .tasktext {
	flex: 1 1 220px;
	display: flex;
	align-items: center;
	align-content: center;
	gap: .25rem .5rem;
	flex-wrap: wrap;
	min-inline-size: 0;
	overflow: visible;
	word-break: normal;
	font-size: .875rem;
	font-weight: 400;
	line-height: 1.45;
}
.task .tasktext .task-link {
	color: var(--text-strong);
	font-weight: 500;
	text-decoration: none;
}
.task .tasktext .task-project {font-size: .74rem; font-weight: 400; color: var(--text-light);}
.task .dueDate, .task .dueDate time {font-size: .75rem; font-style: normal;}
.task .project-task-icon {font-size: .76rem; color: var(--text-light);}
.task .favorite {inline-size: 1.8rem; block-size: 1.8rem;}
.task :deep(.fancy-checkbox) {padding-inline-end: .7rem;}
.task :deep(.fancy-checkbox input) {accent-color: var(--primary);}
.task .tasktext.done .task-link {text-decoration: line-through; color: var(--text-light);}
@media screen and (max-width: $tablet) {
	.task {padding: .65rem .7rem; min-block-size: 3.15rem;}
	.task .tasktext {flex-basis: 160px;}
}

.task {
	display: flex;
	flex-wrap: wrap;
	padding: .4rem;
	transition: background-color $transition;
	align-items: center;
	cursor: pointer;
	border-radius: $radius;
	border: 2px solid transparent;

	&:hover {
		background-color: var(--grey-100);
	}

	&:has(*:focus-visible), &:focus {
		box-shadow: 0 0 0 2px hsla(var(--primary-hsl), 0.5);

		a.task-link {
			box-shadow: none;
		}
	}

	@supports not selector(:focus-within) {
		:focus {
			box-shadow: 0 0 0 2px hsla(var(--primary-hsl), 0.5);

			a.task-link {
				box-shadow: none;
			}
		}
	}

	.tasktext,
	&.tasktext {
		text-overflow: ellipsis;
		word-wrap: break-word;
		word-break: break-word;
		display: -webkit-box;
		hyphens: auto;
		-webkit-line-clamp: 4;
		-webkit-box-orient: vertical;
		overflow: hidden;

		flex: 1 0 50%;

	}

	.dueDate {
		display: inline-block;
		margin-inline-start: 5px;

		&:focus-visible {
			box-shadow: none;

			time {
				box-shadow: 0 0 0 1px hsla(var(--primary-hsl), 0.5);
				border-radius: 3px;
			}
		}
	}

	&[data-is-overdue] .dueDate {
		color: var(--danger-text);
	}

	.task-project {
		inline-size: auto;
		color: var(--grey-400);
		font-size: .9rem;
		white-space: nowrap;
	}

	.tasktext :deep(.color-bubble),
	.tasktext :deep(.avatar-wrapper),
	.tasktext :deep(.labels .tag) {
		vertical-align: middle;
		transform: translateY(-2px);
	}

	.avatar {
		border-radius: 50%;
		vertical-align: bottom;
		margin-inline-start: 5px;
		block-size: 27px;
		inline-size: 27px;
	}

	.project-task-icon {
		margin-inline-start: 6px;

		&:not(:first-of-type) {
			margin-inline-start: 8px;
		}

	}

	a {
		color: var(--text);
		transition: color ease $transition-duration;

		&:hover {
			color: var(--grey-900);
		}
	}

	.favorite {
		opacity: 1;
		text-align: center;
		inline-size: 27px;
		transition: opacity $transition, color $transition;
		border-radius: $radius;

		&:hover {
			color: var(--warning);
		}

		&.is-favorite {
			opacity: 1;
			color: var(--warning);
		}
	}

	@media(hover: hover) and (pointer: fine) {
		& .favorite {
			opacity: 0;
		}

		&:hover .favorite {
			opacity: 1;
		}
	}

	.favorite:focus {
		opacity: 1;
	}

	:deep(.fancy-checkbox) {
		block-size: 18px;
		padding-block-start: 0;
		padding-inline-end: .5rem;

		span {
			display: none;
		}

		// Extend the hit target to >=44x44 without affecting layout (WCAG 2.5.5).
		.base-checkbox__label {
			position: relative;

			&::before {
				content: '';
				position: absolute;
				inset-block-start: 50%;
				inset-inline-start: 50%;
				min-block-size: 44px;
				min-inline-size: 44px;
				block-size: 100%;
				inline-size: 100%;
				transform: translate(-50%, -50%);
			}
		}
	}

	.tasktext.done {
		text-decoration: line-through;
		color: var(--grey-500);
	}

	span.parent-tasks {
		color: var(--grey-500);
		inline-size: auto;
	}

	.show-project .parent-tasks {
		padding-inline-start: .25rem;
	}

	.remove {
		color: var(--danger);
	}

	input[type='checkbox'] {
		vertical-align: middle;
	}

	.settings {
		float: inline-end;
		inline-size: 24px;
		cursor: pointer;
	}

	&.loader-container.is-loading:after {
		inset-block-start: calc(50% - 1rem);
		inset-inline-start: calc(50% - 1rem);
		inline-size: 2rem;
		block-size: 2rem;
		border-inline-start-color: var(--grey-300);
		border-block-end-color: var(--grey-300);
	}
}

.subtask-nested {
	margin-inline-start: 1.75rem;
}

:deep(.popup) {
	border-radius: $radius;
	background-color: var(--white);
	box-shadow: var(--shadow-lg);
	color: var(--text);

	&.is-open {
		padding: 1rem;
		border: 1px solid var(--grey-200);
	}
}
</style>
