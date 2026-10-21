<template>
	<main class="organization-center">
		<header class="organization-header">
			<h1>组织任务中心</h1>
			<p>实例管理员视图 · 全部团队、项目、负责人及任务</p>
		</header>
		<div class="organization-filters">
			<label>团队
				<select v-model.number="teamId">
					<option :value="0">全部团队</option>
					<option v-for="team in teams" :key="team.id" :value="team.id">{{ team.name }}</option>
				</select>
			</label>
			<label>项目
				<select v-model.number="projectId">
					<option :value="0">全部项目</option>
					<option v-for="project in projects" :key="project.id" :value="project.id">{{ project.title }}</option>
				</select>
			</label>
			<label>负责人
				<select v-model.number="assigneeId">
					<option :value="0">全部负责人</option>
					<option v-for="person in people" :key="person.id" :value="person.id">{{ person.name || person.username }}</option>
				</select>
			</label>
			<label>完成情况
				<select v-model="done">
					<option value="all">全部任务</option>
					<option value="false">未完成</option>
					<option value="true">已完成</option>
				</select>
			</label>
			<label>搜索<input v-model.trim="search" type="search" placeholder="任务标题" /></label>
		</div>
		<div class="organization-toolbar">
			<p>共 {{ taskResult?.total ?? 0 }} 条任务</p>
			<div class="organization-view-switch" role="group" aria-label="视图">
				<button type="button" :aria-pressed="view === 'list'" @click="view='list'">列表</button>
				<button type="button" :aria-pressed="view === 'gantt'" @click="view='gantt'">甘特图</button>
			</div>
		</div>
		<p v-if="tasksQuery.isPending.value">正在获取全实例任务…</p>
		<p v-else-if="tasksQuery.isError.value" role="alert">组织任务读取失败：{{ tasksQuery.error.value }}</p>
		<p v-else-if="taskResult?.items.length === 0">没有符合条件的任务。</p>
		<div v-else-if="view === 'list'" class="organization-list">
			<div v-for="task in taskResult?.items" :key="task.id" class="organization-row">
				<span class="organization-complete">{{ task.done ? '✓' : '○' }}</span>
				<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">{{ task.title }}</RouterLink>
				<span>{{ task.project_name }}</span>
				<span>{{ task.done ? '已完成' : '未完成' }}</span>
			</div>
		</div>
		<div v-else class="organization-gantt">
			<p v-if="ganttTasks.length === 0">当前页的任务没有有效起止日期。</p>
			<div v-for="task in ganttTasks" :key="task.id" class="organization-gantt-row">
				<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">{{ task.title }}</RouterLink>
				<div class="organization-track"><div class="organization-bar" :style="barStyle(task)" /></div>
			</div>
		</div>
		<div class="organization-pagination">
			<button type="button" :disabled="page <= 1" @click="page--">上一页</button>
			<span>第 {{ page }} / {{ totalPages }} 页</span>
			<button type="button" :disabled="page >= totalPages" @click="page++">下一页</button>
		</div>
		<section class="organization-team-summary">
			<h2>全部团队</h2>
			<div v-for="team in teams" :key="team.id" class="organization-team-row">
				<button type="button" @click="teamId = team.id; page = 1">{{ team.name }}</button>
				<span>{{ team.members }} 人 · {{ team.projects }} 个共享项目</span>
			</div>
		</section>
	</main>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {adminProjectsList, adminUsersList} from '@/client/generated'
import {client} from '@/client/generated/client.gen'
import {getToken} from '@/helpers/auth'

type OrganizationTask = {
	id: number
	title: string
	project_id: number
	project_name: string
	done: boolean
	priority: number
	start_date?: string | null
	end_date?: string | null
	due_date?: string | null
}
type OrganizationTeam = {id: number, name: string, members: number, projects: number}
type Page<T> = {items: T[], total: number, page: number, per_page: number}
const page = ref(1)
const teamId = ref(0)
const projectId = ref(0)
const assigneeId = ref(0)
const done = ref('all')
const search = ref('')
const view = ref<'list' | 'gantt'>('list')

async function getAdminData<T>(path: string, params: Record<string, string>): Promise<Page<T>> {
	const token = getToken()
	if (!token) throw new Error('请先登录')
	const root = client.getConfig().baseUrl ?? '/api/v2'
	const searchParams = new URLSearchParams(params)
	const response = await fetch(`${root}/admin/organization/${path}?${searchParams.toString()}`, {
		headers: {Authorization: `Bearer ${token}`},
		credentials: 'same-origin',
	})
	if (!response.ok) throw new Error(`HTTP ${response.status}`)
	return response.json() as Promise<Page<T>>
}

const teamsQuery = useQuery(() => ({
	queryKey: ['admin', 'organization', 'teams'],
	queryFn: async () => {
		const first = await getAdminData<OrganizationTeam>('teams', {page: '1', per_page: '100'})
		const items = [...first.items]
		const pages = Math.ceil(first.total / 100)
		for (let page = 2; page <= pages; page++) {
			const next = await getAdminData<OrganizationTeam>('teams', {page: String(page), per_page: '100'})
			items.push(...next.items)
		}
		return items
	},
}))
const tasksQuery = useQuery(() => ({
	queryKey: ['admin', 'organization', 'tasks', page.value, teamId.value, projectId.value, assigneeId.value, done.value, search.value],
	queryFn: () => getAdminData<OrganizationTask>('tasks', {
		page: String(page.value),
		per_page: '50',
		...(teamId.value > 0 ? {team_id: String(teamId.value)} : {}),
		...(projectId.value > 0 ? {project_id: String(projectId.value)} : {}),
		...(assigneeId.value > 0 ? {assignee_id: String(assigneeId.value)} : {}),
		...(done.value === 'all' ? {} : {done: done.value}),
		...(search.value ? {q: search.value} : {}),
	}),
}))
const taskResult = computed(() => tasksQuery.data.value)
const teams = computed(() => teamsQuery.data.value ?? [])
const projectsQuery = useQuery(() => ({
	queryKey: ['admin', 'organization', 'projects'],
	queryFn: async () => (await adminProjectsList({query: {page: 1, per_page: 100}})).data.items ?? [],
}))
const peopleQuery = useQuery(() => ({
	queryKey: ['admin', 'organization', 'people'],
	queryFn: async () => (await adminUsersList({query: {page: 1, per_page: 100}})).data.items ?? [],
}))
const projects = computed(() => projectsQuery.data.value ?? [])
const people = computed(() => peopleQuery.data.value ?? [])

const totalPages = computed(() => Math.max(1, Math.ceil((taskResult.value?.total ?? 0) / 50)))
watch([teamId, projectId, assigneeId, done, search], () => {page.value = 1})
function taskSpan(task: OrganizationTask): [number, number] | null {
	const start = Date.parse(task.start_date || '')
	const end = Date.parse(task.end_date || task.due_date || '')
	if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return null
	return [start, end]
}
const ganttTasks = computed(() => (taskResult.value?.items ?? []).filter(task => taskSpan(task) !== null))
const bounds = computed(() => {
	const ranges = ganttTasks.value.map(task => taskSpan(task)!)
	return {min: Math.min(...ranges.map(x => x[0])), max: Math.max(...ranges.map(x => x[1]))}
})
function barStyle(task: OrganizationTask) {
	const span = taskSpan(task)
	if (!span) return {}
	const width = Math.max(86400000, bounds.value.max - bounds.value.min)
	return {
		marginInlineStart: Math.max(0, (span[0]-bounds.value.min)/width*100)+'%',
		width: Math.max(1, (span[1]-span[0])/width*100)+'%',
	}
}
</script>
<style scoped lang="scss">
.organization-center {max-inline-size:1460px; margin-inline:auto; padding:1.25rem;}
.organization-header {padding:1rem 1.25rem; border:1px solid var(--card-border-color); border-radius:10px; background:var(--white); margin-block-end:1rem;}
.organization-header h1 {font-size:1.5rem; font-weight:600;}
.organization-header p {color:var(--text-light); font-size:.85rem;}
.organization-filters {display:grid; grid-template-columns:repeat(auto-fit,minmax(170px,1fr)); gap:.8rem; background:var(--white); border:1px solid var(--card-border-color); border-radius:10px; padding:1rem;}
.organization-filters label {display:flex; flex-direction:column; gap:.35rem; font-size:.8rem;}
.organization-filters input,.organization-filters select {padding:.5rem; border:1px solid var(--card-border-color); border-radius:6px; background:var(--white); color:var(--text);}
.organization-toolbar,.organization-pagination {display:flex; align-items:center; justify-content:space-between; gap:1rem; padding:1rem 0;}
.organization-view-switch {display:flex; border:1px solid var(--card-border-color); border-radius:6px; overflow:hidden;}
.organization-view-switch button,.organization-pagination button {padding:.4rem .7rem; border:1px solid var(--card-border-color); border-radius:5px; background:var(--white); color:var(--text); cursor:pointer;}
.organization-view-switch button[aria-pressed="true"] {background:var(--grey-100); font-weight:600;}
.organization-pagination button:disabled {opacity:.5; cursor:not-allowed;}
.organization-list,.organization-gantt,.organization-team-summary {background:var(--white); border:1px solid var(--card-border-color); border-radius:10px; padding:.6rem 1rem;}
.organization-row {display:grid; grid-template-columns:1.5rem minmax(140px,2fr) minmax(90px,1fr) 5rem; align-items:center; gap:1rem; padding:.8rem .4rem; border-bottom:1px solid var(--card-border-color); font-size:.86rem;}
.organization-complete {color:var(--text-light);}
.organization-gantt-row {display:grid; grid-template-columns:minmax(110px,200px) 1fr; gap:1rem; align-items:center; margin:.7rem 0;}
.organization-track {height:1.3rem; background:var(--grey-50); border-radius:4px;}
.organization-bar {height:100%; background:var(--primary); border-radius:4px;}
.organization-team-summary h2 {font-size:1rem; font-weight:600; margin:.3rem 0 .8rem;}
.organization-team-row {display:flex; justify-content:space-between; padding:.5rem; border-bottom:1px solid var(--card-border-color);}
.organization-team-row span {color:var(--text-light); font-size:.8rem;}
@media(max-width:720px){.organization-row{grid-template-columns:1.5rem 1fr;}.organization-row span:nth-last-child(-n+2){display:none;}}
</style>
