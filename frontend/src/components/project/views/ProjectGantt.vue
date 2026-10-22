<template>
	<ProjectWrapper
		class="project-gantt"
		:is-loading-project="isLoadingProject"
		:project-id
		:view-id
	>
		<template #default>
			<Card :has-content="false">
				<div class="gantt-options">
					<FormField
						id="range"
						:label="$t('misc.dateRange')"
					>
						<template #default="{ id }">
							<DateRangeInput
								:id="id"
								v-model="dateRange"
								:placeholder="$t('misc.dateRange')"
							/>
						</template>
					</FormField>
					<XButton v-if="hasProjectRange" variant="secondary" @click="applyProjectDateRange">
						按项目起止时间
					</XButton>
					<div
						v-if="!hasDefaultFilters"
						class="field"
					>
						<span
							class="label"
							aria-hidden="true"
						>Reset</span>
						<div class="control">
							<XButton @click="setDefaultFilters">
								Reset
							</XButton>
						</div>
					</div>
					<FancyCheckbox
						v-model="filters.showTasksWithoutDates"
						is-block
					>
						{{ $t('task.show.noDates') }}
					</FancyCheckbox>
				</div>
			</Card>

			<div class="gantt-chart-container">
				<Card
					:has-content="false"
					:padding="false"
					class="has-overflow"
				>
					<GanttChart
						:filters="filters"
						:tasks="tasks"
						:is-loading="isLoading"
						:default-task-start-date="defaultTaskStartDate"
						:default-task-end-date="defaultTaskEndDate"
						@update:task="updateTask"
					/>
					<TaskForm
						v-if="canWrite"
						@createTask="addGanttTask"
					/>
				</Card>
			</div>
		</template>
	</ProjectWrapper>
</template>

<script setup lang="ts">
import {computed, toRefs, watch, ref} from 'vue'
import type {RouteLocationNormalized} from 'vue-router'

import {useCurrentProject} from '@/composables/useCurrentProject'

import DateRangeInput from '@/components/input/DateRangeInput.vue'
import ProjectWrapper from '@/components/project/ProjectWrapper.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import FormField from '@/components/input/FormField.vue'

import GanttChart from '@/components/gantt/GanttChart.vue'
import {useGanttFilters} from '../../../views/project/helpers/useGanttFilters'
import {PERMISSIONS} from '@/constants/permissions'

import type {DateISO} from '@/types/DateISO'
import type {Task as ITask} from '@/client/generated'

const props = defineProps<{
	isLoadingProject: boolean,
	projectId: number,
	route: RouteLocationNormalized
	viewId: number
}>()


const {currentProject} = useCurrentProject()
const canWrite = computed(() =>
	typeof currentProject.value?.max_permission === 'number' &&
	currentProject.value.max_permission > PERMISSIONS.READ,
)

const {route, projectId, viewId} = toRefs(props)
const {
	filters,
	hasDefaultFilters,
	setDefaultFilters,
	tasks,
	isLoading,
	addTask,
	updateTask,
} = useGanttFilters(route, projectId, viewId)

type ProjectPlanningDates = {start_date?: string | null, end_date?: string | null, due_date?: string | null}
const projectDates = computed(() => currentProject.value as (typeof currentProject.value & ProjectPlanningDates))
const hasProjectRange = computed(() => Boolean(projectDates.value?.start_date && (projectDates.value?.end_date || projectDates.value?.due_date)))
function applyProjectDateRange() {
	const start = projectDates.value?.start_date
	const end = projectDates.value?.end_date || projectDates.value?.due_date
	if (!start || !end) return
	const from = new Date(start)
	const to = new Date(end)
	if (Number.isNaN(from.getTime()) || Number.isNaN(to.getTime()) || from > to) return
	filters.value.dateFrom = from.toISOString()
	filters.value.dateTo = to.toISOString()
}
const initializedProject = ref<number | null>(null)
watch([currentProject, projectId], ([project, id]) => {
	if (!project || initializedProject.value === id || !hasProjectRange.value) return
	initializedProject.value = id
	if (route.value.query.dateFrom || route.value.query.dateTo) return
	applyProjectDateRange()
}, {immediate: true})

const DEFAULT_DATE_RANGE_DAYS = 7

const today = new Date()
const defaultTaskStartDate: DateISO = new Date(today.setHours(0, 0, 0, 0)).toISOString()
const defaultTaskEndDate: DateISO = new Date(new Date(
	today.getFullYear(),
	today.getMonth(),
	today.getDate() + DEFAULT_DATE_RANGE_DAYS,
).setHours(23, 59, 0, 0)).toISOString()

async function addGanttTask(title: ITask['title']) {
	return await addTask({
		title,
		project_id: filters.value.projectId,
		start_date: defaultTaskStartDate,
		end_date: defaultTaskEndDate,
	})
}

const dateRange = computed({
	get: () => ({
		start: new Date(filters.value.dateFrom),
		end: new Date(filters.value.dateTo),
	}),
	set({start, end}: {start: Date, end: Date}) {
		Object.assign(filters.value, {dateFrom: start.toISOString(), dateTo: end.toISOString()})
	},
})
</script>

<style lang="scss" scoped>
.gantt-chart-container {
	padding-block-end: 1rem;
	position: relative;
	z-index: 0;
}

.gantt-options {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-block-end: 1rem;

	@media screen and (max-width: $tablet) {
		flex-direction: column;
	}
}

:global(.link-share-view:not(.has-background)) .gantt-options {
	border: none;
	box-shadow: none;

	.card-content {
		padding: .5rem;
	}
}

.field {
	margin-block-end: 0;
	inline-size: 33%;

	&:not(:last-child) {
		padding-inline-end: .5rem;
	}

	@media screen and (max-width: $tablet) {
		inline-size: 100%;
		max-inline-size: 100%;
		margin-block-start: .5rem;
		padding-inline-end: 0 !important;
	}

	&, .input {
		font-size: .8rem;
	}

	.select,
	.select select {
		block-size: auto;
		inline-size: 100%;
		font-size: .8rem;
	}

	.label {
		font-size: .9rem;
	}
}
</style>
