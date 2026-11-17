<template>
	<ProjectWrapper
		class="project-list"
		:is-loading-project="isLoadingProject"
		:project-id="projectId"
		:view-id
	>
		<template #header>
			<div class="filter-container">
				<SortPopup
					v-model="sortByParam"
				/>
				<FilterPopup
					v-if="!isSavedFilterProject(project)"
					v-model="params"
					:view-id="viewId"
					:project-id="projectId"
					@update:modelValue="loadTasks()"
				/>
			</div>
		</template>

		<template #default>
			<div class="workspace-list-toolbar d-print-none">
				<div class="workspace-list-summary"><strong>{{ tasks.length }}</strong> 条当前页任务 <span>· 点击预览查看详情</span></div>
				<button type="button" class="workspace-density-toggle" :aria-pressed="compactRows" @click="compactRows = !compactRows">
					{{ compactRows ? '标准间距' : '紧凑显示' }}
				</button>
			</div>
			<div
				:class="{ 'is-loading': showLoading, 'rows-compact': compactRows }"
				class="loader-container is-max-width-desktop list-view"
			>
				<Card
					:padding="false"
					:has-content="false"
					class="has-overflow"
				>
					<AddTask
						v-if="!project?.is_archived && canWrite"
						ref="addTaskRef"
						class="list-view__add-task d-print-none"
						@tasksAdded="updateTaskList"
					/>

					<Nothing v-if="ctaVisible && tasks.length === 0 && !loading">
						{{ $t('project.list.empty') }}
						<ButtonLink
							v-if="(project?.id ?? 0) > 0 && canWrite"
							@click="focusNewTaskInput()"
						>
							{{ $t('project.list.newTaskCta') }}
						</ButtonLink>
					</Nothing>

					<draggable
						v-if="tasks && tasks.length > 0"
						v-model="tasks"
						:group="{name: 'tasks', put: false}"
						:disabled="!canDragTasks || !isPositionSorting"
						item-key="id"
						tag="ul"
						:component-data="{
							class: {
								tasks: true,
								'dragging-disabled': !canDragTasks || !isPositionSorting
							},
							type: 'transition-group'
						}"
						:animation="100"
						:handle="dragHandle"
						:delay-on-touch-only="!isTouchDevice"
						:delay="isTouchDevice ? 0 : 1000"
						ghost-class="task-ghost"
						@start="handleDragStart"
						@end="saveTaskPosition"
					>
						<template #item="{element: t, index}">
							<SingleTaskInProject
								:ref="(el) => setTaskRef(el as InstanceType<typeof SingleTaskInProject> | null, index)"
								:show-list-color="false"
								:can-mark-as-done="canWrite || isPseudoProject"
								:the-task="t"
								:all-tasks="allTasks"
								@taskUpdated="updateTasks"
							>
								<span
									v-if="canDragTasks && isPositionSorting"
									class="icon handle"
								>
									<Icon icon="grip-lines" />
								</span>
								<button
									type="button"
									class="workspace-row-preview d-print-none"
									:aria-label="`预览任务：${t.title}`"
									@click.stop="openQuickPreview(t.id, $event)"
								>预览</button>
							</SingleTaskInProject>
						</template>
					</draggable>

					<Pagination
						:total-pages="totalPages"
						:current-page="currentPage"
					/>
				</Card>
			</div>
		</template>
	</ProjectWrapper>
	<Teleport to="body">
		<div v-if="previewTask" class="workspace-preview-overlay" @click.self="closeQuickPreview">
			<aside
					ref="previewPanel"
					class="workspace-preview-panel"
					role="dialog"
					aria-modal="true"
					:aria-label="`任务预览：${previewTask.title}`"
					tabindex="-1"
					@keydown.esc.prevent.stop="closeQuickPreview"
					@keydown.tab="trapQuickPreviewFocus"
				>
					<div class="workspace-preview-head">
						<div><span class="workspace-preview-overline">任务快速预览 · #{{ previewTask.id }}</span><h2>{{ previewTask.title }}</h2></div>
						<button type="button" class="workspace-preview-close" aria-label="关闭预览" @click="closeQuickPreview">×</button>
					</div>
					<dl class="workspace-preview-fields">
						<dt>状态</dt><dd><span class="workspace-preview-status" :class="{'is-done': previewTask.done}">{{ previewTask.done ? '已完成' : '待处理' }}</span></dd>
						<dt>负责人</dt><dd>{{ (previewTask.assignees ?? []).map(a => a.name || a.username).join('、') || '尚未指派' }}</dd>
						<dt>截止日期</dt><dd>{{ previewTask.due_date ? previewDateText(previewTask.due_date) : '未设置' }}</dd>
						<dt>优先级</dt><dd>{{ previewPriorityText(previewTask.priority) }}</dd>
						<dt>标签</dt><dd>
							<span v-if="!(previewTask.labels?.length)">无标签</span>
							<span v-for="label in previewTask.labels ?? []" :key="label.id" class="workspace-preview-tag">{{ label.title }}</span>
						</dd>
						<dt>附件</dt><dd>{{ previewTask.attachments?.length ?? 0 }} 个</dd>
						<dt>完成进度</dt><dd><progress class="workspace-preview-progress" :value="Math.round((previewTask.percent_done ?? 0) * 100)" max="100">{{ Math.round((previewTask.percent_done ?? 0) * 100) }}%</progress></dd>
					</dl>
					<section v-if="previewPlainDescription" class="workspace-preview-description"><h3>描述摘录</h3><p>{{ previewPlainDescription }}</p></section>
					<p class="workspace-preview-note">评论、附件、工时和历史记录请在完整任务详情中查看与编辑。</p>
					<div class="workspace-preview-footer">
						<button type="button" class="workspace-preview-secondary" @click="closeQuickPreview">返回列表</button>
						<RouterLink :to="{name: 'task.detail', params: {id: previewTask.id}}" class="workspace-preview-primary" @click="closeQuickPreview">打开完整详情</RouterLink>
					</div>
			</aside>
		</div>
	</Teleport>
</template>


<script setup lang="ts">
import {useUpdateTaskPositionMutation} from '@/client/queries/taskMutations'
import {ref, computed, nextTick, onMounted, onBeforeUnmount, watch, toRef} from 'vue'
import {draggableFor} from '@/components/base/Draggable'

import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import ButtonLink from '@/components/misc/ButtonLink.vue'
import AddTask from '@/components/tasks/AddTask.vue'
import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'
import FilterPopup from '@/components/project/partials/FilterPopup.vue'
import Nothing from '@/components/misc/Nothing.vue'
import Pagination from '@/components/misc/Pagination.vue'
import SortPopup from '@/components/project/partials/SortPopup.vue'

import {useTaskList} from '@/composables/useTaskList'
import {useTaskDragToProject} from '@/composables/useTaskDragToProject'
import {useCurrentProject} from '@/composables/useCurrentProject'
import {shouldShowTaskInListView} from '@/composables/useTaskListFiltering'
import {PERMISSIONS as Permissions} from '@/constants/permissions'
import {calculateItemPosition} from '@/helpers/calculateItemPosition'
import type {TaskResponse} from '@/client/queries/tasks'
import {isSavedFilterProject} from '@/client/queries/projects'

import {useBaseStore} from '@/stores/base'
import {useTaskDragState} from '@/composables/useTaskDragState'
import {useDelayedLoading} from '@/composables/useDelayedLoading'


const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	viewId: number,
}>()

const draggable = draggableFor<TaskResponse>()

const projectId = toRef(props, 'projectId')

defineOptions({name: 'List'})

const ctaVisible = ref(false)
const compactRows = ref(false)
const previewTaskId = ref<number | null>(null)
const previewPanel = ref<HTMLElement | null>(null)
let previewTrigger: HTMLElement | null = null

function closeQuickPreview() {
	previewTaskId.value = null
	nextTick(() => previewTrigger?.focus())
}

function previewDateText(value: string | null | undefined): string {
	if (!value) return '未设置'
	const date = new Date(value)
	return Number.isFinite(date.getTime()) && date.getFullYear() >= 2000 ? date.toLocaleDateString() : '未设置'
}

function previewPriorityText(priority: number | null | undefined): string {
	return ['未设置', '低', '普通', '高', '紧急', '最高'][priority ?? 0] ?? String(priority)
}

function trapQuickPreviewFocus(event: KeyboardEvent) {
	const panel = previewPanel.value
	if (!panel) return
	const elements = Array.from(panel.querySelectorAll<HTMLElement>('button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled])'))
	if (!elements.length) {
		event.preventDefault()
		panel.focus()
		return
	}
	const first = elements[0]
	const last = elements[elements.length - 1]
	if (event.shiftKey && (document.activeElement === first || document.activeElement === panel)) {
		event.preventDefault()
		last.focus()
	} else if (!event.shiftKey && document.activeElement === last) {
		event.preventDefault()
		first.focus()
	}
}

function openQuickPreview(id: number, event: MouseEvent) {
	previewTrigger = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
	previewTaskId.value = id
	void nextTick(() => previewPanel.value?.focus())
}


const drag = ref(false)

const {
	tasks: allTasks,
	loading,
	totalPages,
	currentPage,
	loadTasks,
	params,
	sortByParam,
} = useTaskList(
	() => projectId.value,
	() => props.viewId,
	{position: 'asc'},
	() => projectId.value === -1
		? ['comment_count', 'is_unread']
		: ['subtasks', 'comment_count', 'is_unread'],
)
const showLoading = useDelayedLoading(loading)

const positionMutation = useUpdateTaskPositionMutation()

const dragTasks = ref<TaskResponse[] | null>(null)
const tasks = computed({
	get: () => dragTasks.value ?? allTasks.value.filter(task => shouldShowTaskInListView(task, allTasks.value)),
	set: value => { dragTasks.value = value },
})
const previewTask = computed(() => tasks.value.find(task => task.id === previewTaskId.value) ?? null)
const previewPlainDescription = computed(() =>
	(previewTask.value?.description ?? '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim().slice(0, 320),
)
watch([projectId, () => props.viewId], () => {
	dragTasks.value = null
	previewTaskId.value = null
})

const isPositionSorting = computed(() => 'position' in sortByParam.value)

const baseStore = useBaseStore()
const {setDraggedTask} = useTaskDragState()
const {handleTaskDropToProject} = useTaskDragToProject()
const {currentProject: project} = useCurrentProject()

const canWrite = computed(() => {
	return typeof project.value?.max_permission === 'number' &&
		project.value.max_permission > Permissions.READ &&
		project.value.id > 0
})

const isPseudoProject = computed(() => isSavedFilterProject(project.value) || project.value?.id === -1)

onMounted(async () => {
	await nextTick()
	ctaVisible.value = true
})

const canDragTasks = computed(() => canWrite.value || isSavedFilterProject(project.value))

const isTouchDevice = ref(false)
if (typeof window !== 'undefined') {
	isTouchDevice.value = !window.matchMedia('(hover: hover) and (pointer: fine)').matches
}
const dragHandle = computed(() => isTouchDevice.value ? '.handle' : undefined)

const addTaskRef = ref<typeof AddTask | null>(null)

function focusNewTaskInput() {
	addTaskRef.value?.focusTaskInput()
}

function updateTaskList() {
	baseStore.setHasTasks(true)
}

function updateTasks() {
	if (projectId.value < 0) void loadTasks()
}

function handleDragStart(e: { item: HTMLElement }) {
	drag.value = true
	dragTasks.value = [...tasks.value]
	const taskId = parseInt(e.item.dataset.taskId ?? '', 10)
	const task = tasks.value.find(t => t.id === taskId)

	if (task) {
		setDraggedTask(task)
	}
}

async function saveTaskPosition(e: { originalEvent?: MouseEvent, to: HTMLElement, from: HTMLElement, item: HTMLElement, newIndex: number }) {
	drag.value = false
	try {

		// Check if dropped on a sidebar project
		const {moved} = await handleTaskDropToProject(e, (task) => {
			tasks.value = tasks.value.filter(t => t.id !== task.id)
		})

		if (moved) {
			return
		}

		// If dropped outside this list
		if (e.to !== e.from) {
			return
		}

		// e.newIndex is a DOM index: it counts elements still leaving the transition group, so it can
		// point past the last task. The list is already reordered here, so resolve the task by its id.
		const movedTaskId = parseInt(e.item.dataset.taskId ?? '', 10)
		const newIndex = tasks.value.findIndex(t => t.id === movedTaskId)

		if (newIndex === -1) {
			return
		}

		const taskBefore = tasks.value[newIndex - 1] ?? null
		const taskAfter = tasks.value[newIndex + 1] ?? null

		const position = calculateItemPosition(
			taskBefore !== null ? taskBefore.position : null,
			taskAfter !== null ? taskAfter.position : null,
		)

		await positionMutation.mutateAsync({
			position,
			project_view_id: props.viewId,
			taskId: movedTaskId,
		})
	} catch { /* Mutation reports the error. */ } finally {
		dragTasks.value = null
	}
}

const taskRefs = ref<(InstanceType<typeof SingleTaskInProject> | null)[]>([])
const focusedIndex = ref(-1)

function setTaskRef(el: InstanceType<typeof SingleTaskInProject> | null, index: number) {
	if (el === null) {
		delete taskRefs.value[index]
	} else {
		taskRefs.value[index] = el
	}
}

function focusTask(index: number) {
	if (index < 0 || index >= tasks.value.length) {
		return
	}

	const taskRef = taskRefs.value[index]

	focusedIndex.value = index
	taskRef?.focus()
}

function handleListNavigation(e: KeyboardEvent) {
	if (previewTaskId.value !== null) return
	if (e.target instanceof HTMLElement && (e.target.closest('input, textarea, select, [contenteditable="true"]'))) {
		return
	}

	if (e.code === 'KeyJ') {
		e.preventDefault()
		focusTask(Math.min(focusedIndex.value + 1, tasks.value.length - 1))
		return
	}

	if (e.code === 'KeyK') {
		e.preventDefault()
		if (focusedIndex.value === -1) {
			focusTask(tasks.value.length - 1)
			return
		}

		if (focusedIndex.value === 0) {
			addTaskRef.value?.focusTaskInput()
			focusedIndex.value = -1
			return
		}

		focusTask(Math.max(focusedIndex.value - 1, 0))
		return
	}

	if (e.code === 'Enter') {
		if (e.isComposing) {
			return
		}

		// Links and buttons activate natively on Enter; leave them alone
		if (e.target instanceof HTMLElement && e.target.closest('a, button, [role="button"]')) {
			return
		}

		// Only act when a row was focused via J/K roving navigation
		if (focusedIndex.value < 0) {
			return
		}

		e.preventDefault()
		taskRefs.value[focusedIndex.value]?.click(e)
	}
}

onMounted(() => {
	document.addEventListener('keydown', handleListNavigation)
})

onBeforeUnmount(() => {
	document.removeEventListener('keydown', handleListNavigation)
})
</script>

<style lang="scss" scoped>
.workspace-list-toolbar {
	display: flex; justify-content: space-between; align-items: center; gap: .75rem;
	padding: .55rem .85rem; margin-block-end: .6rem;
	border: 1px solid var(--card-border-color); border-radius: 8px; background: var(--white);
	font-size: .8rem;
}
.workspace-list-summary { color: var(--text-light); }
.workspace-list-summary strong { color: var(--text-strong); font-variant-numeric: tabular-nums; }
.workspace-density-toggle, .workspace-row-preview {
	font: inherit; color: var(--text-light); background: transparent;
	border: 1px solid var(--card-border-color); border-radius: 6px;
	cursor: pointer; padding: .35rem .65rem; white-space: nowrap;
}
.workspace-density-toggle:hover, .workspace-row-preview:hover { color: var(--primary); background: var(--grey-50); }
.workspace-density-toggle:focus-visible, .workspace-row-preview:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
.workspace-row-preview { font-size: .72rem; opacity: 0; margin-inline-start: auto; }
:deep(.single-task:hover) .workspace-row-preview,
:deep(.single-task:focus-within) .workspace-row-preview { opacity: 1; }
.rows-compact :deep(.single-task) { padding-block: .22rem; min-block-size: 2.25rem; }
.rows-compact :deep(.single-task .tasktext) { font-size: .81rem; }
@media (hover: none) { .workspace-row-preview { opacity: 1; } }
@media screen and (max-width: $tablet) {
	.workspace-list-toolbar { flex-wrap: wrap; padding: .5rem .65rem; }
	.workspace-list-summary span { display: none; }
	.workspace-row-preview { opacity: 1; }
}
.workspace-preview-overlay {
	position: fixed; inset: 0; z-index: 150;
	display: flex; justify-content: flex-end;
	background: rgba(16, 24, 40, .35);
}
.workspace-preview-panel {
	inline-size: min(100%, 460px); block-size: 100%; overflow-y: auto;
	background: var(--white); color: var(--text);
	box-shadow: -10px 0 34px rgba(0,0,0,.12);
	display: flex; flex-direction: column; outline: none;
}
.workspace-preview-head {
	display: flex; justify-content: space-between; gap: .75rem;
	padding: 1.5rem; border-block-end: 1px solid var(--card-border-color);
}
.workspace-preview-overline { color: var(--text-light); font-size: .72rem; letter-spacing: .06em; }
.workspace-preview-head h2 { font-size: 1.2rem; font-weight: 700; line-height: 1.35; color: var(--text-strong); margin: .55rem 0 0; overflow-wrap: anywhere; }
.workspace-preview-close { background: transparent; border: 0; color: var(--text-light); font-size: 1.65rem; padding: .1rem .5rem; cursor: pointer; align-self: flex-start; }
.workspace-preview-fields { display: grid; grid-template-columns: 6.2rem minmax(0, 1fr); gap: 1rem; padding: 1.5rem; font-size: .86rem; }
.workspace-preview-fields dt { color: var(--text-light); }
.workspace-preview-fields dd { min-inline-size: 0; margin: 0; overflow-wrap: anywhere; }
.workspace-preview-status { display: inline-flex; padding: .23rem .6rem; border-radius: 999px; background: var(--grey-100); }
.workspace-preview-status.is-done { color: #1a7f37; background: color-mix(in srgb, #1a7f37 12%, var(--white)); }
.workspace-preview-tag { display: inline-block; border: 1px solid var(--card-border-color); border-radius: 5px; padding: .15rem .4rem; margin: 0 .3rem .3rem 0; }
.workspace-preview-description { padding: 0 1.5rem 1.25rem; }
.workspace-preview-description h3 { font-size: .78rem; font-weight: 700; color: var(--text-light); margin-block-end: .5rem; }
.workspace-preview-description p { font-size: .85rem; line-height: 1.6; overflow-wrap: anywhere; color: var(--text); margin: 0; }
.workspace-preview-progress { inline-size: 100%; block-size: .55rem; accent-color: var(--primary); }
.workspace-preview-note { color: var(--text-light); font-size: .81rem; line-height: 1.6; margin: auto 1.5rem 1rem; }
.workspace-preview-footer { display: flex; justify-content: flex-end; gap: .6rem; border-block-start: 1px solid var(--card-border-color); padding: 1rem 1.5rem; }
.workspace-preview-footer a, .workspace-preview-footer button { display: inline-flex; align-items: center; padding: .6rem .85rem; border-radius: 6px; font-size: .85rem; text-decoration: none; cursor: pointer; }
.workspace-preview-secondary { background: var(--white); border: 1px solid var(--card-border-color); color: var(--text); }
.workspace-preview-primary { background: var(--primary); border: 1px solid var(--primary); color: var(--white); }
.workspace-preview-footer :is(a, button):focus-visible, .workspace-preview-close:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
@media (max-width: 480px) {
	.workspace-preview-head, .workspace-preview-fields { padding: 1rem; }
	.workspace-preview-fields { grid-template-columns: 5.2rem minmax(0, 1fr); gap: .8rem; }
	.workspace-preview-footer { padding: 1rem; }
}

.filter-container {
	display: flex;
	align-items: center;
	gap: .5rem;

	:deep(.popup) {
		max-inline-size: 300px;
	}
}

.tasks {
	padding: .45rem .65rem .75rem;
	list-style: none;
}
.tasks :deep(.single-task) {max-inline-size: 100%;}
.list-view__add-task {border-bottom: 1px solid var(--card-border-color);}
.list-view :deep(.card) {background: transparent; border: 0; box-shadow: none;}

.task-ghost {
	border-radius: $radius;
	background: var(--grey-100);
	border: 2px dashed var(--grey-300);

	* {
		opacity: 0;
	}
}

.list-view__add-task {
	padding: 1rem 1rem 0;
}

.link-share-view .card {
	border: none;
	box-shadow: none;
}

:deep(.single-task .handle) {
	cursor: grab;
	margin-inline-end: .25rem;
	color: var(--grey-400);
}

@media (hover: hover) and (pointer: fine) {
	:deep(.single-task .handle) {
		display: none;
	}
}

:deep(.tasks:not(.dragging-disabled) .single-task) {
	cursor: grab;
	-webkit-touch-callout: none;
	user-select: none;
	touch-action: manipulation;

	&:active {
		cursor: grabbing;
	}
}

.list-view {
	padding-block-end: 1rem;

	:deep(.card) {
		margin-block-end: 0;
	}
}
</style>
