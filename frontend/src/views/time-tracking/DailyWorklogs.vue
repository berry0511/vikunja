<template>
	<div class="daily-worklogs">
		<h1 class="title">每日工时</h1>
		<p class="has-text-grey mbs-2">每天填写实际工时，按项目和人员统计，支持 CSV 导出。</p>

		<section class="box">
			<div class="worklog-grid">
				<div class="field">
					<label class="label" for="worklog-project">项目</label>
					<div class="select is-fullwidth">
						<select id="worklog-project" v-model.number="projectId">
							<option :value="0" disabled>请选择项目</option>
							<option v-for="p in availableProjects" :key="p.id" :value="p.id">{{ p.title }}</option>
						</select>
					</div>
				</div>
				<div class="field">
					<label class="label" for="worklog-from">开始日期</label>
					<input id="worklog-from" v-model="fromDate" type="date" class="input">
				</div>
				<div class="field">
					<label class="label" for="worklog-to">结束日期</label>
					<input id="worklog-to" v-model="toDate" type="date" class="input">
				</div>
				<div class="field">
					<label class="label" for="worklog-person">人员</label>
					<div class="select is-fullwidth">
						<select id="worklog-person" v-model.number="userId">
							<option :value="0">全部（普通成员仅自己）</option>
							<option v-for="p in people" :key="p.id" :value="p.id">{{ p.name }}</option>
						</select>
					</div>
				</div>
			</div>
			<div class="buttons">
				<button type="button" class="button is-primary" :disabled="!projectId" @click="toggleForm">
					{{ showForm ? '收起' : '填写工时' }}
				</button>
				<button type="button" class="button" :disabled="!projectId || !validRange || exporting" @click="exportCSV">
					{{ exporting ? '导出中…' : '导出 CSV' }}
				</button>
			</div>
		</section>

		<section v-if="showForm && projectId" class="box">
			<h2 class="subtitle">{{ editingId ? '编辑工时' : '新增工时' }}</h2>
			<form @submit.prevent="save">
				<div class="worklog-grid">
					<div class="field">
						<label class="label" for="worklog-date">工作日期</label>
						<input id="worklog-date" v-model="workDate" type="date" class="input" required>
					</div>
					<div class="field">
						<label class="label" for="worklog-hours">实际小时数</label>
						<input id="worklog-hours" v-model.number="hours" type="number" inputmode="decimal" class="input" min="0.01" max="24" step="any" required>
					</div>
					<div class="field">
						<label class="label" for="worklog-task">任务 ID（可不填）</label>
						<input id="worklog-task" v-model.number="taskId" type="number" min="0" inputmode="numeric" class="input" placeholder="直接记在项目上">
					</div>
				</div>
				<div class="field">
					<label class="label" for="worklog-description">今天做了什么</label>
					<textarea id="worklog-description" v-model="comment" class="textarea" rows="2" maxlength="2000" placeholder="例如：通信协议联调与回归测试" />
				</div>
				<p v-if="formError" role="alert" class="has-text-danger mbs-2">{{ formError }}</p>
				<div class="buttons">
					<button type="submit" class="button is-primary" :disabled="saving">{{ saving ? '保存中…' : '保存' }}</button>
					<button type="button" class="button" @click="resetForm">取消</button>
				</div>
			</form>
		</section>

		<p v-if="!validRange" class="has-text-danger">请选择有效的日期范围。</p>
		<template v-else-if="projectId">
			<section class="box">
				<h2 class="subtitle">每日汇总</h2>
				<p v-if="summaryQuery.isPending.value">正在载入汇总…</p>
				<template v-if="summaryQuery.data.value">
					<p>累计 <strong>{{ formatMinutes(summaryQuery.data.value.total_minutes) }}</strong>，
						共 {{ summaryQuery.data.value.record_count }} 条工时。</p>
					<div class="table-container">
						<table class="table is-fullwidth is-striped">
							<thead><tr><th>日期</th><th>人员</th><th class="has-text-right">投入</th></tr></thead>
							<tbody>
								<tr v-for="d in summaryQuery.data.value.days" :key="d.work_date + '-' + d.user_id">
									<td>{{ d.work_date }}</td>
									<td>{{ d.username || '#' + d.user_id }}</td>
									<td class="has-text-right">{{ formatMinutes(d.minutes) }}</td>
								</tr>
								<tr v-if="summaryQuery.data.value.days.length === 0">
									<td colspan="3">这个时间段还没有记录。</td>
								</tr>
							</tbody>
						</table>
					</div>
				</template>
			</section>

			<section class="box">
				<h2 class="subtitle">工时明细</h2>
				<p v-if="listQuery.isPending.value">正在载入明细…</p>
				<div class="table-container">
					<table class="table is-fullwidth is-striped">
						<thead><tr><th>日期</th><th>人员</th><th>任务</th><th>工作内容</th><th class="has-text-right">投入</th><th>操作</th></tr></thead>
						<tbody>
							<tr v-for="entry in entries" :key="entry.id">
								<td>{{ entry.work_date.slice(0, 10) }}</td>
								<td>{{ entry.username || '#' + entry.user_id }}</td>
								<td>
									<RouterLink v-if="entry.task_id" :to="{name: 'task.detail', params: {id: entry.task_id}}">
										#{{ entry.task_id }}
									</RouterLink>
									<span v-else>项目工作</span>
								</td>
								<td class="worklog-description">{{ entry.comment }}</td>
								<td class="has-text-right">{{ formatMinutes(entry.minutes) }}</td>
								<td>
									<div v-if="entry.user_id === currentUserId" class="buttons are-small">
										<button type="button" class="button" @click="edit(entry)">修改</button>
										<button type="button" class="button is-danger is-light" :disabled="deleting" @click="remove(entry)">删除</button>
									</div>
								</td>
							</tr>
							<tr v-if="entries.length === 0"><td colspan="6">没有匹配记录。</td></tr>
						</tbody>
					</table>
				</div>
				<div v-if="(listQuery.data.value?.total_pages ?? 0) > 1" class="buttons">
					<button type="button" class="button" :disabled="page <= 1" @click="page--">上一页</button>
					<span class="button is-static">{{ page }} / {{ listQuery.data.value?.total_pages }}</span>
					<button type="button" class="button" :disabled="page >= (listQuery.data.value?.total_pages ?? 0)" @click="page++">下一页</button>
				</div>
			</section>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useRoute} from 'vue-router'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {
	downloadWorklogs,
	useWorklogs,
	useWorklogSummary,
	useCreateWorklog,
	useUpdateWorklog,
	useDeleteWorklog,
	type Worklog,
} from '@/client/queries/worklogs'

function dateString(date: Date): string {
	return [date.getFullYear(), String(date.getMonth() + 1).padStart(2, '0'), String(date.getDate()).padStart(2, '0')].join('-')
}

const route = useRoute()
const projects = useProjects()
const auth = useAuthStore()
const availableProjects = computed(() => projects.projectsArray.filter(p => p.id > 0 && !p.is_archived))
const projectId = ref(Number(route.query.project_id) || 0)
const taskId = ref(Number(route.query.task_id) || 0)
const today = dateString(new Date())
const fromDate = ref(dateString(new Date(new Date().getFullYear(), new Date().getMonth(), 1)))
const toDate = ref(today)
const userId = ref(0)
const page = ref(1)
const validRange = computed(() => !!fromDate.value && !!toDate.value && fromDate.value <= toDate.value)
const filter = computed(() => ({
	project_id: validRange.value ? projectId.value : 0,
	user_id: userId.value,
	from_date: fromDate.value,
	to_date: toDate.value,
}))
const peopleFilter = computed(() => ({...filter.value, user_id: 0}))
const listQuery = useWorklogs(filter, page)
const summaryQuery = useWorklogSummary(filter)
const everyoneQuery = useWorklogSummary(peopleFilter)
const entries = computed(() => listQuery.data.value?.items ?? [])
const currentUserId = computed(() => auth.info?.id ?? 0)
const people = computed(() => {
	const names = new Map<number, string>()
	for (const d of everyoneQuery.data.value?.days ?? []) {
		names.set(d.user_id, d.username || '#' + d.user_id)
	}
	return [...names].map(([id, name]) => ({id, name})).sort((a, b) => a.name.localeCompare(b.name))
})
watch([projectId, fromDate, toDate, userId], () => {page.value = 1})
watch(projectId, () => {userId.value = 0})

const showForm = ref(!!route.query.task_id)
const editingId = ref(0)
const workDate = ref(today)
const hours = ref(1)
const comment = ref('')
const formError = ref('')
const saving = ref(false)
const deleting = ref(false)
const exporting = ref(false)
const create = useCreateWorklog()
const update = useUpdateWorklog()
const del = useDeleteWorklog()

function formatMinutes(minutes: number): string {
	return (minutes / 60).toFixed(2) + ' 小时'
}
function toggleForm() {
	if (showForm.value) resetForm()
	else showForm.value = true
}
function resetForm() {
	showForm.value = false
	editingId.value = 0
	taskId.value = 0
	hours.value = 1
	workDate.value = today
	comment.value = ''
	formError.value = ''
}
function edit(entry: Worklog) {
	projectId.value = entry.project_id
	editingId.value = entry.id
	workDate.value = entry.work_date.slice(0, 10)
	taskId.value = entry.task_id
	hours.value = entry.minutes / 60
	comment.value = entry.comment
	formError.value = ''
	showForm.value = true
}
async function save() {
	formError.value = ''
	const minutes = Math.round(Number(hours.value) * 60)
	if (!workDate.value || !Number.isFinite(minutes) || minutes < 1 || minutes > 1440 ||
		!Number.isInteger(Number(taskId.value)) || taskId.value < 0) {
		formError.value = '请填写有效的日期、任务 ID 和 0～24 小时的投入时间。'
		return
	}
	const body = {
		project_id: projectId.value,
		task_id: taskId.value || 0,
		work_date: workDate.value + 'T00:00:00Z',
		minutes,
		comment: comment.value,
	}
	saving.value = true
	try {
		if (editingId.value) await update.mutateAsync({id: editingId.value, ...body})
		else await create.mutateAsync(body)
		resetForm()
	} catch {
		formError.value = '保存失败。请确认任务属于所选项目且你具有访问权限。'
	} finally {
		saving.value = false
	}
}
async function remove(entry: Worklog) {
	if (!confirm('确定删除这条工时记录吗？')) return
	deleting.value = true
	try {
		await del.mutateAsync(entry.id)
	} catch {
		formError.value = '删除失败，请重试。'
	} finally {
		deleting.value = false
	}
}
async function exportCSV() {
	exporting.value = true
	try {
		const file = await downloadWorklogs(filter.value)
		const url = URL.createObjectURL(file)
		const link = document.createElement('a')
		link.href = url
		link.download = 'worklogs-' + fromDate.value + '-' + toDate.value + '.csv'
		document.body.appendChild(link)
		link.click()
		link.remove()
		URL.revokeObjectURL(url)
	} catch {
		formError.value = '导出失败，请确认所选日期和项目权限。'
	} finally {
		exporting.value = false
	}
}
</script>

<style scoped lang="scss">
.daily-worklogs {
	max-inline-size: 1200px;
	margin-inline: auto;
	padding: 1rem;
}
.worklog-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 1rem;
	margin-block-end: 1rem;
}
.worklog-description {
	max-inline-size: 22rem;
	overflow-wrap: anywhere;
}
</style>
