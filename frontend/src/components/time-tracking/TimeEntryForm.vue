<template>
	<form
		ref="formEl"
		v-cy="'timeEntryForm'"
		class="time-entry-form"
		@submit.prevent="saveEntry"
	>
		<div
			v-if="taskId === undefined"
			class="field-columns"
		>
			<div class="field">
				<label class="label">{{ $t('task.attributes.project') }}</label>
				<ProjectSearch v-model="selectedProject" />
			</div>

			<div class="field">
				<label class="label">{{ $t('timeTracking.form.task') }}</label>
				<Multiselect
					v-model="selectedTask"
					:placeholder="$t('timeTracking.form.taskSearch')"
					:loading="taskQuery.isFetching.value"
					:search-results="foundTasks"
					label="title"
					@focusin="refreshTaskResults"
					@search="findTasks"
				>
					<template #searchResult="{option}">
						{{ option.title }}
					</template>
				</Multiselect>
				<p v-if="taskSelectionError" class="help is-danger" role="alert">{{ taskSelectionError }}</p>
			</div>
		</div>

		<div class="field">
			<label class="label">{{ $t('task.comment.comment') }}</label>
			<input
				v-model="comment"
				v-cy="'timeEntryComment'"
				class="input"
				type="text"
				:placeholder="$t('timeTracking.form.commentPlaceholder')"
			>
		</div>

		<div class="field is-grouped from-to-row">
			<div class="control is-expanded">
				<label class="label">{{ $t('input.datepickerRange.from') }}</label>
				<Datepicker
					v-model="from"
					:show-shortcuts="false"
				/>
			</div>
			<div class="control is-expanded">
				<label class="label">{{ $t('input.datepickerRange.to') }}</label>
				<Datepicker
					v-model="to"
					:show-shortcuts="false"
					:empty-label="$t('misc.notSet')"
				/>
			</div>
			<div class="control">
				<BaseButton
					v-tooltip="$t('timeTracking.form.smartFill')"
					v-cy="'smartFill'"
					class="smart-fill"
					:aria-label="$t('timeTracking.form.smartFill')"
					@click="smartFill"
				>
					<Icon :icon="['far', 'clock']" />
				</BaseButton>
			</div>
		</div>

		<div class="field form-actions">
			<template v-if="isEditing">
				<XButton
					v-cy="'updateTimeEntry'"
					:aria-disabled="!canSubmit || undefined"
					:loading="isSaving"
					@click="saveEntry"
				>
					{{ $t('timeTracking.form.update') }}
				</XButton>
				<XButton
					variant="secondary"
					@click="cancelEdit"
				>
					{{ $t('misc.cancel') }}
				</XButton>
			</template>
			<template v-else>
				<XButton
					v-cy="'saveTimeEntry'"
					:aria-disabled="!canSubmit || undefined"
					:loading="isSaving"
					@click="saveEntry"
				>
					{{ $t('timeTracking.form.save') }}
				</XButton>
				<XButton
					v-cy="'startTimer'"
					variant="secondary"
					:aria-disabled="!canSubmit || undefined"
					:loading="isSaving"
					@click="startTimer"
				>
					{{ $t('timeTracking.form.startTimer') }}
				</XButton>
			</template>
		</div>
	</form>
</template>

<script setup lang="ts">
import {ref, computed, watch, nextTick} from 'vue'

import BaseButton from '@/components/base/BaseButton.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import Datepicker from '@/components/input/Datepicker.vue'
import ProjectSearch from '@/components/tasks/partials/ProjectSearch.vue'

import {useTasks} from '@/composables/useTasks'
import {ensureTask} from '@/client/queries/tasks'
import {smartFillStart} from '@/helpers/time/smartFillStart'
import {useCreateTimeEntryMutation, useUpdateTimeEntryMutation} from '@/client/queries/timeEntries'
import {tasksRead} from '@/client/generated'
import type {UpdateTimeEntryInput} from '@/client/queries/timeEntries'
import type {TimeEntryWritable} from '@/client/generated'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {useAuthStore} from '@/stores/auth'
import {useProjects} from '@/composables/useProjects'

import type {ProjectResponse} from '@/client/queries/projects'
import type {TaskResponse} from '@/client/queries/tasks'
import type {TimeEntryResponse as ITimeEntry} from '@/client/queries/timeEntries'

const props = withDefaults(defineProps<{
	// When set, the entry is locked to this task and the project/task pickers are hidden.
	taskId?: number
	// When set, the form edits this entry (Update + Cancel) instead of creating.
	entry?: ITimeEntry | null
	// Entries the smart-clock looks at to continue from the last one's end.
	recentEntries?: ITimeEntry[]
}>(), {
	taskId: undefined,
	entry: undefined,
	recentEntries: () => [],
})

const emit = defineEmits<{
	saved: []
	cancel: []
}>()

const createMutation = useCreateTimeEntryMutation()
const updateMutation = useUpdateTimeEntryMutation()
const authStore = useAuthStore()
const projectList = useProjects()

const isEditing = computed(() => props.entry != null)

const formEl = ref<HTMLFormElement | null>(null)
const selectedProject = ref<ProjectResponse | null>(null)
const selectedTask = ref<TaskResponse | null>(null)
const start = ref<{date: Date | null, chosen: boolean}>({
	date: new Date(),
	chosen: false,
})
const from = computed({
	get: () => start.value.date,
	set: date => {
		start.value = {
			date,
			chosen: true,
		}
	},
})
const to = ref<Date | null>(null)
const comment = ref('')
const isSaving = ref(false)

// Task and project are mutually exclusive (XOR) — selecting one clears the other,
// so applyTarget never picks a stale target the user has since changed.
const taskSelectionError = ref('')
watch(selectedTask, task => {
	taskSelectionError.value = ''
	if (task !== null) {
		selectedProject.value = null
	}
})
watch(selectedProject, project => {
	if (project !== null) {
		selectedTask.value = null
	}
})

const taskSearch = ref('')
const taskQuery = useTasks(
	() => ({project: selectedProject.value?.id, params: {q: taskSearch.value, sort_by: ['done']}}),
	{enabled: () => taskSearch.value !== ''},
)
// Soft-deleted tasks must never be offered as targets. The server's
// soft-delete condition filters active searches, while this also protects
// against stale cached results and tasks returned with deleted_at.
const foundTasks = computed(() => taskQuery.tasks.value.filter(task => !task.deleted_at))
function findTasks(query: string) { taskSearch.value = query }
function refreshTaskResults() {
	if (taskSearch.value !== '') void taskQuery.refetch()
}

// Search lists may have been cached before another user deleted the task.
// Revalidate the chosen task against the server *before* creating an entry;
// the backend permission check remains authoritative too.
async function ensureSelectedTaskStillExists(): Promise<boolean> {
	const task = selectedTask.value
	if (task === null) return true
	try {
		const {data} = await tasksRead({path: {task: task.id}})
		if (data?.id === task.id && !data.deleted_at) return true
	} catch {
		// Deleted, inaccessible or unavailable. Fail closed and ask to reselect.
	}
	if (selectedTask.value?.id === task.id) {
		selectedTask.value = null
		taskSelectionError.value = 'This task is no longer available. Please select an existing task.'
	}
	return false
}


const canSubmit = computed(() =>
	// In edit mode the entry already has a valid container; an update that sends
	// neither keeps it, so don't block submit if the prefill lookup failed.
	isEditing.value || props.taskId !== undefined || selectedTask.value !== null || selectedProject.value !== null,
)

function smartFill() {
	from.value = smartFillStart(
		props.recentEntries,
		authStore.settings.frontend_settings.time_tracking_default_start ?? '09:00',
		new Date(),
	)
	to.value = new Date()
}

// Whichever of task / project is set lands on the payload (XOR — enforced by canSubmit).
function applyTarget(payload: TimeEntryWritable) {
	if (props.taskId !== undefined) {
		payload.task_id = props.taskId
	} else if (selectedTask.value !== null) {
		payload.task_id = selectedTask.value.id
	} else if (selectedProject.value !== null) {
		payload.project_id = selectedProject.value.id
	}
}

function buildPayload(includeEnd: boolean): TimeEntryWritable {
	const payload: TimeEntryWritable = {
		comment: comment.value,
		start_time: (from.value ?? new Date()).toISOString(),
	}
	applyTarget(payload)
	// Saving a manual entry always has an end (an empty "To" means "until now");
	// only the Start-timer path omits it to create a running timer.
	if (includeEnd) {
		payload.end_time = (to.value ?? new Date()).toISOString()
	}
	return payload
}

function reset() {
	selectedTask.value = null
	selectedProject.value = null
	comment.value = ''
	start.value = {
		date: new Date(),
		chosen: false,
	}
	to.value = null
}

// Prefill from the entry being edited; a null entry returns the form to create mode.
watch(() => props.entry, async (entry, _previous, onCleanup) => {
	let active = true
	onCleanup(() => { active = false })
	if (entry == null) {
		reset()
		return
	}
	comment.value = entry.comment
	from.value = parseDateOrNull(entry.start_time)
	to.value = parseDateOrNull(entry.end_time)
	// Bring the form into view — the edit button may be far down the list.
	await nextTick()
	if (!active) return
	formEl.value?.scrollIntoView({behavior: 'smooth', block: 'center'})
	if (props.taskId !== undefined) {
		return
	}
	if (entry.task_id > 0) {
		selectedProject.value = null
		selectedTask.value = null
		try {
			const task = await ensureTask(entry.task_id)
			if (active && selectedProject.value === null && selectedTask.value === null) {
				selectedTask.value = task
			}
		} catch {
			return
		}
	} else if (entry.project_id > 0) {
		selectedTask.value = null
		selectedProject.value = projectList.projects[entry.project_id] ?? null
	}
}, {immediate: true})

function draftIdentity() {
	return JSON.stringify([
		props.entry?.id,
		props.taskId,
		selectedTask.value?.id,
		selectedProject.value?.id,
		from.value,
		to.value,
		comment.value,
	])
}

async function submit(includeEnd: boolean) {
	if (!canSubmit.value || isSaving.value) {
		return
	}
	isSaving.value = true
	const draft = draftIdentity()
	try {
		if (!(await ensureSelectedTaskStillExists())) return
		if (draftIdentity() !== draft) return
		const payload = buildPayload(includeEnd)
		// An untouched start would otherwise be when the form loaded, not click time.
		if (!includeEnd && !start.value.chosen) {
			payload.start_time = new Date().toISOString()
		}
		await createMutation.mutateAsync(payload)
		if (draftIdentity() !== draft) return
		reset()
		emit('saved')
	} catch {
		return
	} finally {
		isSaving.value = false
	}
}

async function submitUpdate() {
	const entry = props.entry
	if (!canSubmit.value || isSaving.value || entry == null) {
		return
	}
	isSaving.value = true
	const draft = draftIdentity()
	try {
		const payload: UpdateTimeEntryInput = {
			id: entry.id,
			comment: comment.value,
			start_time: from.value?.toISOString() ?? entry.start_time,
			// A running entry stays running (null); a completed one can't be reopened,
			// so keep its end if "To" was cleared (the API rejects clearing it).
			end_time: to.value?.toISOString() ?? entry.end_time,
			task_id: 0,
			project_id: 0,
		}
		applyTarget(payload)
		await updateMutation.mutateAsync(payload)
		if (draftIdentity() !== draft) return
		emit('saved')
	} catch {
		return
	} finally {
		isSaving.value = false
	}
}

const saveEntry = () => (isEditing.value ? submitUpdate() : submit(true))
const startTimer = () => submit(false)
function cancelEdit() {
	emit('cancel')
}
</script>

<style lang="scss" scoped>
.field-columns {
	display: flex;
	gap: 1rem;

	> .field {
		flex: 1;
		min-inline-size: 0;
	}
}

.from-to-row {
	align-items: flex-end;
}

.smart-fill {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	block-size: 2.5em;
	inline-size: 2.5em;
	border-radius: $radius;
	color: var(--primary);
	transition: background-color $transition;

	&:hover {
		background-color: var(--grey-100);
	}
}

.form-actions {
	display: flex;
	gap: .5rem;
}
</style>
