<template>
	<div class="team-workspace">
		<header class="team-workspace-heading">
			<div>
				<RouterLink :to="{name: 'teams.index'}">← 团队列表</RouterLink>
				<h1>{{ team?.name ?? '团队工作区' }}</h1>
				<p>{{ team?.description || '团队成员与协作空间' }}</p>
			</div>
			<RouterLink v-if="canManage" class="team-workspace-manage" :to="{name: 'teams.edit', params: {id: teamId}}">管理团队</RouterLink>
		</header>
		<p v-if="teamQuery.isPending.value">正在加载团队…</p>
		<p v-else-if="teamQuery.isError.value" role="alert">无法读取团队，请检查团队成员权限。</p>
		<section v-else class="team-workspace-panel">
			<h2>团队成员 <small>({{ team?.members?.length ?? 0 }})</small></h2>
			<ul class="team-member-list">
				<li v-for="member in team?.members ?? []" :key="member.id">
					<span>{{ member.name || member.username }}</span>
					<span class="team-member-role">{{ member.admin ? '团队管理员' : '成员' }}</span>
				</li>
			</ul>
			<div class="team-project-help">
				<h2>团队共享项目 <small>({{ sharedProjects.length }})</small></h2>
				<p>仅展示明确授权给本团队且当前账号有权访问的项目，不包含团队成员私人任务。</p>
				<p v-if="projectsQuery.isPending.value">正在核对团队项目授权…</p>
				<p v-else-if="projectsQuery.isError.value" role="alert">团队项目查询失败，请稍后重试。</p>
				<p v-else-if="sharedProjects.length === 0">暂时没有可访问的团队共享项目。项目管理员可以在项目共享设置中添加此团队。</p>
				<ul v-else class="team-shared-projects">
					<li v-for="project in sharedProjects" :key="project.id">
						<RouterLink :to="{name: 'project.index', params: {projectId: project.id}}">{{ project.title }}</RouterLink>
						<span>团队权限：{{ project.permission === 2 ? '管理' : project.permission === 1 ? '读写' : '只读' }}</span>
					</li>
				</ul>
				<RouterLink :to="{name: 'projects.index'}">查看所有我有权限的项目 →</RouterLink>
			</div>
			<div class="team-progress-strip" aria-label="当前团队可见任务进度">
				<div><strong>{{ sharedProjects.length }}</strong><span>共享项目</span></div>
				<div><strong>{{ teamTasks.length }}</strong><span>可见任务</span></div>
				<div><strong>{{ completedTeamTasks }}</strong><span>已完成</span></div>
				<div class="team-progress-meter"><span>任务完成比例</span><progress :value="completedTeamTasks" :max="Math.max(1, teamTasks.length)">{{ teamTasks.length ? Math.round(completedTeamTasks / teamTasks.length * 100) : 0 }}%</progress></div>
			</div>
			<div class="team-task-board">
				<div class="team-task-heading">
					<h2>团队任务 <small>({{ teamTasks.length }})</small></h2>
					<div class="team-view-switch" role="group" aria-label="切换团队视图">
						<button type="button" :aria-pressed="view === 'list'" @click="view = 'list'">列表</button>
						<button type="button" :aria-pressed="view === 'gantt'" @click="view = 'gantt'">甘特图</button>
					</div>
				</div>
				<p v-if="tasksQuery.isPending.value">正在读取有权限的任务…</p>
				<p v-else-if="tasksQuery.isError.value" role="alert">任务读取失败，请检查项目访问权限。</p>
				<p v-else-if="teamTasks.length === 0">当前团队共享项目中没有可见任务。</p>
				<ul v-else-if="view === 'list'" class="team-task-list">
					<li v-for="task in teamTasks" :key="task.id">
						<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">{{ task.title }}</RouterLink>
						<span>{{ task.done ? '已完成' : '未完成' }} · {{ sharedProjects.find(p => p.id === task.project_id)?.title }}</span>
					</li>
				</ul>
				<div v-else class="team-gantt">
					<p v-if="ganttTasks.length === 0">没有设置起止日期的团队任务。</p>
					<div v-for="task in ganttTasks" :key="task.id" class="team-gantt-line">
						<RouterLink :to="{name: 'task.detail', params: {id: task.id}}">{{ task.title }}</RouterLink>
						<div class="team-gantt-track"><div class="team-gantt-bar" :style="ganttStyle(task)" /></div>
					</div>
				</div>
			</div>
		</section>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {projectTeamsList} from '@/client/generated'
import {ensureProjects} from '@/client/queries/projects'
import {allTasksQuery} from '@/client/queries/tasks'
import {useAuthStore} from '@/stores/auth'
import {useQuery} from '@tanstack/vue-query'
import {teamQuery as getTeamQuery} from '@/client/queries/teams'

const props = defineProps<{id: string}>()
const teamId = computed(() => Number(props.id))
const teamQuery = useQuery(() => getTeamQuery(teamId.value))
const team = computed(() => teamQuery.data.value)
const authStore = useAuthStore()
const canManage = computed(() => Boolean(authStore.info?.is_admin || team.value?.members?.some(member => member.id === authStore.info?.id && member.admin)))
const projectsQuery = useQuery(() => ({
	queryKey: ['teams', 'workspace', teamId.value, 'shared-projects'],
	enabled: teamQuery.isSuccess.value,
	queryFn: async () => {
		const projects = (await ensureProjects()).projects.filter(project => !project.is_archived)
		const matches: Array<{id: number, title: string, permission: number}> = []
		// Project team listings apply existing project ACL checks server-side.
		// Avoid parallel flooding instances with many projects.
		for (const project of projects) {
			let page = 1
			for (;;) {
				const response = await projectTeamsList({path: {project: project.id}, query: {page, per_page: 100}})
				const team = response.data.items?.find(entry => entry.id === teamId.value)
				if (team) {
					matches.push({id: project.id, title: project.title, permission: team.permission ?? 0})
					break
				}
				if (page >= (response.data.total_pages ?? 1)) break
				page++
			}
		}
		return matches
	},
}))
const sharedProjects = computed(() => projectsQuery.data.value ?? [])
const view = ref<'list' | 'gantt'>('list')
const tasksQuery = useQuery(() => ({
	...allTasksQuery({}),
	enabled: projectsQuery.isSuccess.value,
}))
const teamTasks = computed(() => {
	const ids = new Set(sharedProjects.value.map(project => project.id))
	return (tasksQuery.data.value ?? []).filter(task => ids.has(task.project_id))
})
const completedTeamTasks = computed(() => teamTasks.value.filter(task => task.done).length)
function span(task: {start_date?: string | null, end_date?: string | null, due_date?: string | null}) {
	const start = task.start_date ? new Date(task.start_date) : null
	const end = task.end_date || task.due_date ? new Date(task.end_date || task.due_date || '') : null
	if (!start || !end || !Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime())) return null
	return [start, end] as const
}
const ganttTasks = computed(() => teamTasks.value.filter(task => span(task) !== null))
const chartBounds = computed(() => {
	const dates = ganttTasks.value.flatMap(task => span(task) ?? [])
	const min = Math.min(...dates.map(date => date.getTime()))
	const max = Math.max(...dates.map(date => date.getTime()))
	return {min, width: Math.max(86400000, max - min)}
})
function ganttStyle(task: {start_date?: string | null, end_date?: string | null, due_date?: string | null}) {
	const dates = span(task)
	if (!dates) return {}
	return {
		marginInlineStart: Math.max(0, (dates[0].getTime() - chartBounds.value.min) / chartBounds.value.width * 100) + '%',
		width: Math.max(1, (dates[1].getTime() - dates[0].getTime()) / chartBounds.value.width * 100) + '%',
	}
}


</script>

<style scoped lang="scss">
.team-workspace {max-inline-size: 1100px; margin-inline: auto; padding: 1.2rem;}
.team-workspace-heading {display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:1rem; padding:1rem 1.2rem; margin-block-end:1rem; background:var(--white); border:1px solid var(--card-border-color); border-radius:10px;}
.team-workspace-heading h1 {font-size:1.45rem; font-weight:600; margin:.5rem 0 .2rem;}
.team-workspace-heading p {font-size:.85rem; color:var(--text-light);}
.team-workspace-manage {border:1px solid var(--card-border-color); border-radius:6px; padding:.45rem .75rem;}
.team-workspace-panel {border:1px solid var(--card-border-color); border-radius:10px; padding:1rem 1.2rem; background:var(--white);}
.team-workspace-panel h2 {font-size:1rem; font-weight:600;}
.team-workspace-panel small {color:var(--text-light); font-weight:400;}
.team-member-list {list-style:none; margin: .75rem 0 1.5rem; padding:0;}
.team-member-list li {display:flex; justify-content:space-between; gap:1rem; padding:.65rem 0; border-bottom:1px solid var(--card-border-color);}
.team-member-role {color:var(--text-light); font-size:.8rem;}
.team-project-help {padding-block-start:.75rem;}
.team-project-help p {font-size:.85rem; margin:.7rem 0; color:var(--text-light);}
.team-shared-projects {list-style:none; padding:0; margin:.75rem 0;}
.team-shared-projects li {padding:.65rem 0; border-bottom:1px solid var(--card-border-color); display:flex; justify-content:space-between; gap:.75rem;}
.team-shared-projects span {font-size:.8rem; color:var(--text-light);}
.team-task-board {border-top:1px solid var(--card-border-color); padding-block-start:1rem; margin-block-start:1rem;}
.team-task-heading {display:flex; align-items:center; justify-content:space-between; gap:1rem;}
.team-view-switch {display:flex; border:1px solid var(--card-border-color); border-radius:6px; overflow:hidden;}
.team-view-switch button {padding:.3rem .6rem; cursor:pointer; background:var(--white); color:var(--text); border:0;}
.team-view-switch button[aria-pressed="true"] {background:var(--grey-100); font-weight:600;}
.team-task-list {list-style:none; margin:.75rem 0; padding:0;}
.team-task-list li {display:flex; justify-content:space-between; gap:1rem; padding:.7rem 0; border-bottom:1px solid var(--card-border-color);}
.team-task-list span {font-size:.78rem; color:var(--text-light);}
.team-gantt {margin-block-start:.75rem;}
.team-gantt-line {display:grid; grid-template-columns:12rem 1fr; gap:.75rem; align-items:center; margin-block:.5rem;}
.team-gantt-line a {overflow:hidden; text-overflow:ellipsis; white-space:nowrap;}
.team-gantt-track {background:var(--grey-50); block-size:1.3rem; border-radius:4px;}
.team-gantt-bar {background:var(--primary); block-size:100%; border-radius:4px;}
@media(max-width:720px){.team-workspace{padding:.6rem;}.team-workspace-heading,.team-workspace-panel{padding:.75rem;}.team-task-heading,.team-member-list li,.team-shared-projects li{flex-wrap:wrap;}.team-gantt-line {grid-template-columns:minmax(0,7rem) minmax(0,1fr);}.team-task-list li{flex-direction:column;}}

.team-progress-strip {display:grid;grid-template-columns:repeat(3,minmax(0,1fr)) minmax(130px,2fr);gap:.6rem;margin-block:1.1rem;border:1px solid var(--card-border-color);border-radius:8px;padding:.75rem;background:var(--grey-50);}
.team-progress-strip>div {display:flex;flex-direction:column;justify-content:center;gap:.35rem;min-inline-size:0;}
.team-progress-strip strong {color:var(--text-strong);font-size:1.4rem;font-weight:700;font-variant-numeric:tabular-nums;}
.team-progress-strip span {color:var(--text-light);font-size:.75rem;}
.team-progress-meter progress {inline-size:100%;block-size:.5rem;accent-color:var(--primary);}
.team-task-heading {flex-wrap:wrap;}
.team-view-switch button {font-size:.8rem;min-block-size:2.3rem;}
.team-view-switch button[aria-pressed="true"] {color:var(--primary);background:color-mix(in srgb,var(--primary) 9%,var(--white));}
.team-view-switch button:focus-visible {outline:2px solid var(--primary);outline-offset:-2px;}
.team-task-list li {align-items:center;font-size:.84rem;}
.team-task-list li>a {min-inline-size:0;overflow-wrap:anywhere;color:var(--text-strong);}
.team-task-list li>a:hover {color:var(--primary);}
.team-gantt-track {overflow:hidden;}
.team-workspace-manage {background:var(--white);font-size:.84rem;}
@media(max-width:720px){.team-progress-strip{grid-template-columns:repeat(3,minmax(0,1fr));}.team-progress-meter{grid-column:1/-1;}.team-workspace{max-inline-size:100%;}.team-gantt{overflow-x:auto;-webkit-overflow-scrolling:touch;}}
</style>
