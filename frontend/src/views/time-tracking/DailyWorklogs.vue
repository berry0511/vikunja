<template>
	<div class="daily-report">
		<header class="report-header">
			<div>
				<h1 class="title is-3">工时统计</h1>
				<p class="report-muted">按项目、人员和日期核对每日投入。工时请在对应任务中记录。</p>
			</div>
			<XButton variant="secondary" icon="download" :disabled="!validRange || exporting" :loading="exporting" @click="exportCSV()">
				导出明细
			</XButton>
		</header>

		<Card title="筛选条件">
			<div class="report-filters">
				<div class="field">
					<label class="label" for="daily-report-project">项目</label>
					<FormSelect id="daily-report-project" v-model.number="projectId" :options="projectOptions" />
				</div>
				<div class="field">
					<label class="label" for="daily-report-from">开始日期</label>
					<FormInput id="daily-report-from" v-model="fromDate" type="date" />
				</div>
				<div class="field">
					<label class="label" for="daily-report-to">结束日期</label>
					<FormInput id="daily-report-to" v-model="toDate" type="date" />
				</div>
				<div class="field">
					<label class="label" for="daily-report-user">人员</label>
					<FormSelect id="daily-report-user" v-model.number="userId" :options="peopleOptions" />
				</div>
			</div>
			<div class="report-filter-footer">
				<p class="report-muted">所有工时均关联真实任务；项目成员只能查询自己的记录。</p>
				<XButton type="button" variant="tertiary" @click="resetFilters">重置筛选</XButton>
			</div>
		</Card>

		<div v-if="!validRange" class="report-feedback" role="alert">结束日期不能早于开始日期。</div>
		<template v-else>
			<div class="report-metrics">
				<Card>
					<p class="metric-label">累计投入</p>
					<strong class="metric-value">{{ formatMinutes(summaryQuery.data.value?.total_minutes ?? 0) }}</strong>
				</Card>
				<Card>
					<p class="metric-label">工时记录</p>
					<strong class="metric-value">{{ summaryQuery.data.value?.record_count ?? 0 }} <small>条</small></strong>
				</Card>
				<Card>
					<p class="metric-label">参与人员</p>
					<strong class="metric-value">{{ participantCount }} <small>人</small></strong>
				</Card>
			</div>

			<Card title="每日投入">
				<div v-if="summaryQuery.isPending.value" class="report-loading">正在获取每日汇总…</div>
				<div v-else-if="!summaryQuery.data.value?.days.length" class="report-empty report-empty--in-card">
					当前范围没有工时记录。请打开对应任务，在任务详情填写工时。
				</div>
				<div v-else class="report-table-scroll">
					<table class="report-table">
						<thead>
							<tr><th>日期</th><th>项目</th><th>人员</th><th class="report-number">投入工时</th></tr>
						</thead>
						<tbody>
							<tr v-for="d in summaryQuery.data.value.days" :key="d.work_date + '-' + d.user_id">
								<td>{{ d.work_date }}</td>
								<td>{{ getProjectTitleByReport(d.project_title) }}</td>
								<td>{{ d.username || '#' + d.user_id }}</td>
								<td class="report-number"><strong>{{ formatMinutes(d.minutes) }}</strong></td>
							</tr>
						</tbody>
					</table>
				</div>
				<div class="report-actions">
					<XButton variant="secondary" icon="download" :disabled="exporting" :loading="exporting" @click="exportCSV(true)">导出每日汇总</XButton>
				</div>
			</Card>

			<Card title="工时明细">
				<div v-if="listQuery.isPending.value" class="report-loading">正在获取工时明细…</div>
				<div v-else-if="entries.length === 0" class="report-empty report-empty--in-card">没有匹配的工时记录。</div>
				<div v-else class="report-table-scroll">
					<table class="report-table">
						<thead>
							<tr><th>日期</th><th>项目</th><th>人员</th><th>关联任务</th><th>工作内容</th><th class="report-number">投入工时</th><th>操作</th></tr>
						</thead>
						<tbody>
							<tr v-for="entry in entries" :key="entry.id">
								<td>{{ entry.work_date.slice(0, 10) }}</td>
								<td>{{ getProjectTitleByReport(entry.project_title) }}</td>
								<td>{{ entry.username || '#' + entry.user_id }}</td>
								<td>
									<RouterLink :to="{name: 'task.detail', params: {id: entry.task_id}}" class="report-task-link">
										#{{ entry.task_id }} <Icon icon="external-link-alt" />
									</RouterLink>
								</td>
								<td class="report-comment">{{ entry.comment || '—' }}</td>
								<td class="report-number"><strong>{{ formatMinutes(entry.minutes) }}</strong></td>
								<td>
									<RouterLink v-if="entry.user_id === currentUserId" :to="{name: 'task.detail', params: {id: entry.task_id}}" class="report-task-link">在任务中编辑</RouterLink>
									<span v-else>—</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
				<div v-if="(listQuery.data.value?.total_pages ?? 0) > 1" class="report-pagination">
					<XButton variant="secondary" :disabled="page <= 1" @click="page--">上一页</XButton>
					<span>{{ page }} / {{ listQuery.data.value?.total_pages }}</span>
					<XButton variant="secondary" :disabled="page >= (listQuery.data.value?.total_pages ?? 0)" @click="page++">下一页</XButton>
				</div>
			</Card>
			<p v-if="reportError" class="report-feedback" role="alert">{{ reportError }}</p>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch, onMounted} from 'vue'
import {useRoute} from 'vue-router'
import Card from '@/components/misc/Card.vue'
import FormSelect from '@/components/input/FormSelect.vue'
import FormInput from '@/components/input/FormInput.vue'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {useBaseStore} from '@/stores/base'
import {useTitle} from '@/composables/useTitle'
import {getProjectTitle} from '@/helpers/getProjectTitle'
import {
	downloadWorklogs,
	downloadWorklogSummary,
	useWorklogs,
	useWorklogSummary,
} from '@/client/queries/worklogs'

function dateString(date: Date): string {
	return [date.getFullYear(), String(date.getMonth() + 1).padStart(2, '0'), String(date.getDate()).padStart(2, '0')].join('-')
}
const today = dateString(new Date())
const defaultFrom = dateString(new Date(new Date().getFullYear(), new Date().getMonth(), 1))
const route = useRoute()
const projects = useProjects()
const auth = useAuthStore()
const baseStore = useBaseStore()
useTitle('工时统计')

const projectId = ref(Number(route.query.project_id) || 0)
const taskId = ref(Number(route.query.task_id) || 0)
const fromDate = ref(defaultFrom)
const toDate = ref(today)
const userId = ref(0)
const page = ref(1)
const validRange = computed(() => !!fromDate.value && !!toDate.value && fromDate.value <= toDate.value)
const projectOptions = computed(() => [
	{value: 0, label: '全部项目'},
	...projects.projectsArray
		.filter(p => p.id > 0 && !p.is_archived)
		.map(p => ({value: p.id, label: getProjectTitle(p)})),
])
const filter = computed(() => ({
	project_id: validRange.value ? projectId.value : -1,
	task_id: taskId.value,
	user_id: userId.value,
	from_date: fromDate.value,
	to_date: toDate.value,
}))
const everyoneFilter = computed(() => ({...filter.value, user_id: 0}))
const listQuery = useWorklogs(filter, page)
const summaryQuery = useWorklogSummary(filter)
const peopleQuery = useWorklogSummary(everyoneFilter)
const entries = computed(() => listQuery.data.value?.items ?? [])
const currentUserId = computed(() => auth.info?.id ?? 0)
const peopleOptions = computed(() => {
	const names = new Map<number, string>()
	for (const day of peopleQuery.data.value?.days ?? []) names.set(day.user_id, day.username || '#' + day.user_id)
	return [{value: 0, label: '全部人员 / 仅本人'}, ...[...names].map(([value, label]) => ({value, label}))]
})
const participantCount = computed(() => new Set((summaryQuery.data.value?.days ?? []).map(day => day.user_id)).size)
function getProjectTitleByReport(title: string): string { return title === 'Inbox' ? '收件箱' : title }
const exporting = ref(false)
const reportError = ref('')
watch([projectId, taskId, fromDate, toDate, userId], () => {page.value = 1; reportError.value = ''})
watch(projectId, () => {userId.value = 0; taskId.value = 0})
onMounted(() => {baseStore.setCurrentProject(null)})

function formatMinutes(minutes: number): string {
	return (minutes / 60).toFixed(2) + ' 小时'
}
function resetFilters() {
	fromDate.value = defaultFrom
	toDate.value = today
	userId.value = 0
	taskId.value = 0
}
async function exportCSV(summary = false) {
	reportError.value = ''
	exporting.value = true
	try {
		const file = summary ? await downloadWorklogSummary(filter.value) : await downloadWorklogs(filter.value)
		const url = URL.createObjectURL(file)
		const link = document.createElement('a')
		link.href = url
		link.download = (summary ? 'worklogs-daily-' : 'worklogs-') + fromDate.value + '-' + toDate.value + '.csv'
		document.body.appendChild(link)
		link.click()
		link.remove()
		URL.revokeObjectURL(url)
	} catch {
		reportError.value = '工时导出失败，请检查项目权限和日期范围。'
	} finally {
		exporting.value = false
	}
}
</script>

<style scoped lang="scss">
.daily-report {max-inline-size: 1200px; margin-inline: auto; padding: clamp(.8rem, 3vw, 1.75rem);}
.report-header {display: flex; justify-content: space-between; align-items: center; gap: 1rem; margin-block-end: 1.5rem;}
.report-header .title {margin: 0 0 .5rem; color: var(--text-strong);}
.report-muted {color: var(--text-light); font-size: .9rem; line-height: 1.5;}
.report-filters {display: grid; grid-template-columns: repeat(auto-fit, minmax(185px, 1fr)); gap: 1rem;}
.report-filters .field {min-inline-size: 0;}
.report-filter-footer {display: flex; justify-content: space-between; align-items: center; gap: 1rem; flex-wrap: wrap; margin-block-start: .6rem;}
.report-metrics {display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1rem; margin-block: 1.25rem;}
.report-metrics :deep(.card) {block-size: 100%;}
.metric-label {font-size: .9rem; color: var(--text-light); margin-block-end: .65rem;}
.metric-value {display: block; color: var(--text-strong); font-size: clamp(1.3rem, 2.5vw, 2rem); line-height: 1.3;}
.metric-value small {font-size: .8rem; color: var(--text-light); font-weight: 400;}
.report-table-scroll {max-inline-size: 100%; overflow-x: auto;}
.report-table {inline-size: 100%; border-collapse: collapse; min-inline-size: 520px; font-size: .9rem;}
.report-table thead {background: var(--grey-50); color: var(--text-light);}
.report-table th {font-weight: 600; text-align: start; padding: .75rem .8rem; white-space: nowrap;}
.report-table td {border-block-start: 1px solid var(--card-border-color); padding: .85rem .8rem; color: var(--text); vertical-align: top;}
.report-table tbody tr:hover {background-color: var(--grey-50);}
.report-table .report-number {text-align: end; white-space: nowrap;}
.report-table .report-comment {min-inline-size: 180px; overflow-wrap: anywhere;}
.report-task-link {white-space: nowrap;}
.report-actions {display: flex; justify-content: flex-end; padding-block-start: 1rem;}
.report-pagination {display: flex; align-items: center; justify-content: center; gap: 1rem; margin-block-start: 1rem;}
.report-loading {padding: 1.25rem 0; color: var(--text-light);}
.report-feedback {padding: .8rem; margin-block: .75rem; color: var(--danger); border-radius: $radius;}
.report-empty {padding: 3rem 1rem; text-align: center; color: var(--text-light);}
.report-empty h2 {font-size: 1.1rem; font-weight: 600; margin-block: .65rem;}
.report-empty--in-card {padding: 1.5rem;}
@media (max-width: 640px) {
	.report-header {align-items: flex-start; flex-direction: column;}
	.report-metrics {grid-template-columns: 1fr;}
}
</style>
