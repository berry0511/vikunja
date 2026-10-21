<!-- eslint-disable vue/no-v-html -->
<template>
	<Modal
		@close="goBack()"
	>
		<Card
			:title="project?.title"
			class="is-justify-content-start"
			:show-close="true"
			@close="goBack()"
		>
			<div
				v-if="htmlDescription !== ''"
				class="has-text-start"
				data-user-content
				v-html="htmlDescription"
			/>
			<p
				v-else
				class="is-italic"
			>
				{{ $t('project.noDescriptionAvailable') }}
			</p>
			<section v-if="project && projectId > 0" class="project-schedule">
				<h3>项目时间计划</h3>
				<p class="schedule-help">用于设置项目整体时间范围；甘特图可一键按项目日期展示。</p>
				<form class="schedule-fields" @submit.prevent="saveSchedule">
					<label>开始时间<input v-model="startDate" type="date" :disabled="!canWrite || saving" /></label>
					<label>结束时间<input v-model="endDate" type="date" :disabled="!canWrite || saving" /></label>
					<label>截止时间<input v-model="dueDate" type="date" :disabled="!canWrite || saving" /></label>
					<div v-if="canWrite" class="schedule-actions"><button type="submit" :disabled="saving">{{ saving ? '保存中…' : '保存项目时间' }}</button></div>
				</form>
				<p v-if="scheduleError" class="schedule-error" role="alert">{{ scheduleError }}</p>
			</section>
			<ChangeHistory v-if="projectId > 0" kind="project" :entity-id="projectId" />
		</Card>
	</Modal>
</template>

<script lang="ts" setup>
import {computed, ref, watch} from 'vue'
import DOMPurify from 'dompurify'
import {useProjects} from '@/composables/useProjects'
import ChangeHistory from '@/components/history/ChangeHistory.vue'
import {projectsUpdate} from '@/client/generated'
import type {ProjectWritable} from '@/client/generated'
import {createProjectDraft, refreshProject, refreshProjects} from '@/client/queries/projects'
import {PERMISSIONS} from '@/constants/permissions'
import {useGoBack} from '@/composables/useGoBack'
import {useI18n} from 'vue-i18n'

const props = defineProps<{
	projectId: number
}>()

const {t} = useI18n()
const goBack = useGoBack()

// ADD_ATTR re-permits target, which lets a link hand the opened page a live
// window.opener back to this tab. rel must be forced here rather than allowed,
// so a description carrying its own rel can't drop noopener.
DOMPurify.addHook('afterSanitizeAttributes', node => {
	if (node.hasAttribute('target')) {
		node.setAttribute('rel', 'noopener noreferrer')
	}
})

const projectList = useProjects()
const project = computed(() => projectList.projects[props.projectId])
const htmlDescription = computed(() => {
	const description = project.value?.description || ''
	if (description === '') {
		return ''
	}

	if (project.value.id === -1) {
		return t('project.favoriteDescription')
	}

	return DOMPurify.sanitize(description, {ADD_ATTR: ['target']})
})

type PlanningDates = {start_date?: string | null, end_date?: string | null, due_date?: string | null}
const canWrite = computed(() => (project.value?.max_permission ?? 0) > PERMISSIONS.READ)
const startDate = ref('')
const endDate = ref('')
const dueDate = ref('')
const saving = ref(false)
const scheduleError = ref('')
function toDateInput(raw?: string | null): string {
	if (!raw) return ''
	const date = new Date(raw)
	if (Number.isNaN(date.getTime())) return ''
	return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
watch(project, p => {
	const schedule = p as (typeof p & PlanningDates)
	startDate.value = toDateInput(schedule?.start_date)
	endDate.value = toDateInput(schedule?.end_date)
	dueDate.value = toDateInput(schedule?.due_date)
}, {immediate: true})
function toTimestamp(date: string, endOfDay = false): string | null {
	if (!date) return null
	return new Date(date + (endOfDay ? 'T23:59:59' : 'T00:00:00')).toISOString()
}
async function saveSchedule() {
	if (!project.value || !canWrite.value || saving.value) return
	scheduleError.value = ''
	if (startDate.value && endDate.value && startDate.value > endDate.value) {
		scheduleError.value = '结束时间不能早于开始时间'
		return
	}
	if (startDate.value && dueDate.value && startDate.value > dueDate.value) {
		scheduleError.value = '截止时间不能早于开始时间'
		return
	}
	saving.value = true
	try {
		const body = {
			...createProjectDraft(project.value),
			start_date: toTimestamp(startDate.value),
			end_date: toTimestamp(endDate.value, true),
			due_date: toTimestamp(dueDate.value, true),
		}
		await projectsUpdate({path: {id: props.projectId}, body: body as ProjectWritable})
		await Promise.all([refreshProject(props.projectId), refreshProjects()])
	} catch (cause) {
		scheduleError.value = cause instanceof Error ? cause.message : '保存失败'
	} finally {
		saving.value = false
	}
}
</script>

<style scoped lang="scss">
.project-schedule {border-block:1px solid var(--card-border-color); padding-block:1rem; margin-block:1rem;}
.project-schedule h3 {font-size:1rem; font-weight:600; margin-block-end:.25rem;}
.schedule-help {font-size:.78rem; color:var(--text-light); margin-block-end:.85rem;}
.schedule-fields {display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:.75rem;}
.schedule-fields label {font-size:.8rem; font-weight:500; display:flex; flex-direction:column; gap:.35rem;}
.schedule-fields input {padding:.5rem; border:1px solid var(--card-border-color); border-radius:6px; background:var(--white); color:var(--text);}
.schedule-actions {grid-column:1 / -1;}
.schedule-actions button {border:1px solid var(--primary); color:var(--white); background:var(--primary); padding:.45rem .8rem; border-radius:6px; cursor:pointer;}
.schedule-actions button:disabled {opacity:.6; cursor:wait;}
.schedule-error {color:var(--danger); font-size:.8rem; margin-block-start:.5rem;}
@media(max-width: 720px){.schedule-fields {grid-template-columns:1fr;}}
</style>
