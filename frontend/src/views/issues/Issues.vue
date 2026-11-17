<template>
	<section class="issue-center">
		<header class="issue-header">
			<div>
				<p class="issue-overline">QUALITY / BUG TRACKING</p>
				<h1>Bug 管理</h1>
				<p>统一登记、分派、复现与验证项目缺陷；评论、附件、工时和操作历史与对应项目关联。</p>
			</div>
			<XButton :disabled="!writableProjects.length" icon="plus" @click="showCreate = !showCreate">
				{{ showCreate ? '收起登记表' : '报告 Bug' }}
			</XButton>
		</header>

		<Card v-if="showCreate" title="报告 Bug">
			<form class="issue-form" @submit.prevent="createIssue">
				<div class="issue-fields">
					<label>归属项目
						<select v-model.number="createProjectId" required>
							<option :value="0" disabled>选择项目</option>
							<option v-for="p in writableProjects" :key="p.id" :value="p.id">{{ getProjectTitle(p) }}</option>
						</select>
					</label>
					<label>优先级
						<select v-model.number="priority">
							<option :value="1">低</option>
							<option :value="2">一般</option>
							<option :value="3">高</option>
							<option :value="4">紧急</option>
							<option :value="5">最高</option>
						</select>
					</label>
				</div>
			<label>标题
					<input v-model.trim="title" type="text" maxlength="250" required placeholder="一句话描述待处理的问题" />
				</label>
			<label>详细说明
					<textarea v-model="details" rows="3" placeholder="背景、影响范围及补充说明" />
				</label>
			<template>
				<label>环境 / 版本
					<input v-model.trim="environment" type="text" placeholder="例如：Android 15 / App 1.2.3 / H743 V2" />
				</label>
				<label>复现步骤
					<textarea v-model="steps" rows="3" placeholder="1. 操作…&#10;2. 观察…" />
				</label>
				<div class="issue-fields">
					<label>预期结果<textarea v-model="expected" rows="2" /></label>
					<label>实际结果<textarea v-model="actual" rows="2" /></label>
				</div>
			</template>
			<p class="issue-help">提交后可在 Bug 详情继续添加负责人、附件、评论和工时。</p>
			<p v-if="createError" class="issue-error" role="alert">{{ createError }}</p>
			<div class="issue-actions">
				<XButton type="submit" :disabled="creating || !createProjectId || !title" :loading="creating">提交 Bug</XButton>
				<XButton type="button" variant="secondary" @click="showCreate = false">取消</XButton>
			</div>
			</form>
		</Card>

		<Card title="Issues · Bug 列表">
			<div class="issue-filter-bar">
				<label>项目
					<select v-model.number="projectId">
						<option :value="0">全部可访问项目</option>
						<option v-for="p in readableProjects" :key="p.id" :value="p.id">{{ getProjectTitle(p) }}</option>
					</select>
				</label>
				<label>状态
					<select v-model="statusFilter">
						<option value="all">全部状态</option>
						<option v-for="(label, value) in ISSUE_STATUS_LABELS" :key="value" :value="value">{{ label }}</option>
					</select>
				</label>
				<label>搜索
					<input v-model.trim="searchText" type="search" placeholder="标题或Bug编号" />
				</label>
			</div>
			<div class="issue-results-toolbar">
				<span aria-live="polite">当前显示 <strong>{{ shownIssues.length }}</strong> / {{ allIssues.length }} 个 Bug</span>
				<button type="button" class="issue-reset-filters" @click="resetIssueFilters">清除筛选</button>
			</div>
			<div class="issue-metrics">
				<div><strong>{{ allIssues.length }}</strong><span>Bug 总数</span></div>
				<div><strong>{{ allIssues.filter(t => issueStatus(t) === 'working').length }}</strong><span>处理中</span></div>
				<div><strong>{{ allIssues.filter(t => !t.done).length }}</strong><span>未完成</span></div>
				<div><strong>{{ allIssues.filter(t => t.done).length }}</strong><span>已完成</span></div>
			</div>
			<p v-if="issuesQuery.isPending.value" class="issue-help">正在读取Bug…</p>
			<p v-else-if="issuesQuery.isError.value" class="issue-error">Bug 查询失败：{{ getErrorText(issuesQuery.error.value) }}</p>
			<p v-else-if="!shownIssues.length" class="issue-empty">当前条件没有 Bug，可以点击「报告 Bug」登记。</p>
			<div v-else class="issue-table-wrap">
				<table class="issue-table">
					<thead><tr><th>Bug 编号 / 标题</th><th>项目</th><th>优先级</th><th>负责人</th><th>状态</th></tr></thead>
					<tbody>
						<tr v-for="item in shownIssues" :key="item.id">
							<td>
								<RouterLink class="issue-title" :to="{name: 'task.detail', params: {id: item.id}}"><span class="issue-state-mark" :class="`state-${issueStatus(item)}`" aria-hidden="true">{{ item.done ? "✓" : "●" }}</span>{{ item.title }}</RouterLink>
								<span class="issue-id">#{{ item.id }} · {{ projectName(item.project_id) }}</span>
							</td>
							<td>{{ projectName(item.project_id) }}</td>
							
							<td>{{ priorityName(item.priority) }}</td>
							<td>{{ item.assignees.map(a => a.name || a.username).join('、') || '待分派' }}</td>
							<td>
								<select :value="issueStatus(item)" :aria-label="'修改 Bug #' + item.id + ' 状态'" :disabled="changingId === item.id" @change="setIssueStatus(item, $event)">
									<option v-for="(label, value) in ISSUE_STATUS_LABELS" :key="value" :value="value">{{ label }}</option>
								</select>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<p v-if="statusError" class="issue-error" role="alert">{{ statusError }}</p>
			<p class="issue-help">Bug 使用项目已有权限。修改状态需要任务编辑权限，详情继续复用任务评论、附件与历史。</p>
		</Card>
	</section>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useRoute, useRouter} from 'vue-router'
import Card from '@/components/misc/Card.vue'
import {useProjects} from '@/composables/useProjects'
import {getProjectTitle} from '@/helpers/getProjectTitle'
import {allTasksQuery, taskKeys, type TaskResponse} from '@/client/queries/tasks'
import {ensureLabels, labelKeys} from '@/client/queries/labels'
import {labelsCreate, tasksCreate, taskLabelsCreate, patchTasksRead} from '@/client/generated'
import {queryClient} from '@/client/queryClient'
import {getErrorText} from '@/message'
import {
	ISSUE_LABELS,
	ISSUE_STATUS_LABELS,
	issueKind,
	issueStatus,
	statusTaskPatch,
	issueDescription,
	type IssueStatus,
} from '@/helpers/issues'

const router = useRouter()
const route = useRoute()
const selectedProjectId = computed(() => {
	const id = Number(route.query.project_id)
	return Number.isSafeInteger(id) && id > 0 ? id : 0
})
const projects = useProjects()
const readableProjects = computed(() => projects.projectsArray.filter(p => p.id > 0 && !p.is_archived))
const writableProjects = computed(() => readableProjects.value.filter(p => (p.max_permission ?? 0) > 0))
const projectId = ref(selectedProjectId.value)
const createProjectId = ref(selectedProjectId.value)
watch(writableProjects, value => {
	if (!createProjectId.value && value.length) createProjectId.value = value[0].id
}, {immediate: true})
const priority = ref(2)
const title = ref('')
const details = ref('')
const environment = ref('')
const steps = ref('')
const expected = ref('')
const actual = ref('')
const creating = ref(false)
const createError = ref('')
const statusError = ref('')
const showCreate = ref(route.query.create === '1')
watch(selectedProjectId, id => {
	projectId.value = id
	if (id > 0) createProjectId.value = id
	showCreate.value = route.query.create === '1'
})
const statusFilter = ref('all')
const searchText = ref('')
function resetIssueFilters() {
	projectId.value = 0
	statusFilter.value = 'all'
	searchText.value = ''
}

const changingId = ref(0)
const scope = computed(() => ({
	project: projectId.value || null,
	params: {sort_by: ['created', 'id'], order_by: ['desc', 'desc'], filter: ''},
}))
const issuesQuery = useQuery(() => allTasksQuery(scope.value))
const allIssues = computed(() => (issuesQuery.data.value ?? []).filter(item => issueKind(item) === 'bug'))
const shownIssues = computed(() => allIssues.value.filter(item => {
	if (statusFilter.value !== 'all' && issueStatus(item) !== statusFilter.value) return false
	if (!searchText.value) return true
	const q = searchText.value.toLowerCase()
	return item.title.toLowerCase().includes(q) || String(item.id).includes(q)
}))
function projectName(id: number) {
	const project = projects.projectsArray.find(p => p.id === id)
	return project ? getProjectTitle(project) : `#${id}`
}
function priorityName(value: number) {
	return ['无', '低', '一般', '高', '紧急', '最高'][Math.max(0, Math.min(5, value))] ?? '一般'
}
async function createIssue() {
	if (!title.value || !createProjectId.value || creating.value) return
	creating.value = true
	createError.value = ''
	let createdId = 0
	try {
		const labelTitle = ISSUE_LABELS.bug
		const labels = await ensureLabels()
		let label = labels.find(item => item.title === labelTitle)
		if (!label) {
			label = (await labelsCreate({body: {
				title: labelTitle,
				description: 'Bug中心内部使用的类型标签',
				hex_color: 'dc4c53',
			}})).data
			await queryClient.invalidateQueries({queryKey: labelKeys.all})
		}
		if (!label?.id) throw new Error('无法创建Bug类型标签')
		const created = (await tasksCreate({path: {project: createProjectId.value}, body: {
			title: title.value,
			description: issueDescription({
				kind: 'bug', details: details.value, environment: environment.value,
				steps: steps.value, expected: expected.value, actual: actual.value,
			}),
			priority: priority.value,
			done: false,
		}})).data
		if (!created.id) throw new Error('Bug已保存，但没有返回任务编号')
		createdId = created.id
		await taskLabelsCreate({path: {task: createdId}, body: {label_id: label.id}})
		await queryClient.invalidateQueries({queryKey: taskKeys.all})
		showCreate.value = false
		await router.push({name: 'task.detail', params: {id: createdId}})
	} catch (reason) {
		createError.value = createdId
			? `任务 #${createdId} 已创建，但Bug类型标签关联失败。请打开该任务检查标签：${getErrorText(reason)}`
			: getErrorText(reason)
	} finally {
		creating.value = false
	}
}
async function setIssueStatus(task: TaskResponse, event: Event) {
	const target = event.target as HTMLSelectElement
	const old = issueStatus(task)
	statusError.value = ''
	const status = target.value as IssueStatus
	if (status === old) return
	changingId.value = task.id
	try {
		const patch = statusTaskPatch(status)
		await patchTasksRead({
			path: {task: task.id},
			body: [
				{op: 'replace', path: '/done', value: patch.done},
				{op: 'replace', path: '/percent_done', value: patch.percent_done},
			],
		})
		await queryClient.invalidateQueries({queryKey: taskKeys.all})
	} catch (reason) {
		target.value = old
		statusError.value = `更新Bug状态失败：${getErrorText(reason)}`
	} finally { changingId.value = 0 }
}
</script>

<style scoped lang="scss">
.issue-center {max-inline-size: 1400px; margin-inline: auto; padding: clamp(.5rem,1.8vw,1.4rem)}
.issue-header {display:flex; align-items:center; justify-content:space-between; gap:1.5rem; margin-block-end:1.5rem}
.issue-header h1 {font-size:clamp(1.6rem,3vw,2.3rem); font-weight:750; color:var(--text-strong); margin:.4rem 0}
.issue-header p {color:var(--text-light); font-size:.9rem; line-height:1.5}
.issue-header .issue-overline {font-size:.72rem; font-weight:800; letter-spacing:.12em; color:var(--primary)}
.issue-form, .issue-form label {display:flex; flex-direction:column; gap:.45rem}
.issue-form {gap:1.05rem}
.issue-form label, .issue-filter-bar label {font-size:.87rem; font-weight:650; color:var(--text-strong)}
.issue-fields,.issue-filter-bar {display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,195px),1fr)); gap:1rem}
.issue-form input,.issue-form select,.issue-form textarea,.issue-filter-bar input,.issue-filter-bar select,.issue-table select {
	width:100%; padding:.7rem .8rem; background:var(--white); border:1px solid var(--card-border-color);
	border-radius:9px; color:var(--text); font:inherit; font-weight:400;
}
.issue-form input:focus,.issue-form textarea:focus,.issue-filter-bar input:focus,.issue-form select:focus,.issue-table select:focus {
	outline:2px solid var(--primary); outline-offset:1px;
}
.issue-form textarea {resize:vertical; min-block-size:70px}
.issue-actions {display:flex; flex-wrap:wrap; gap:.7rem}
.issue-help {font-size:.8rem; line-height:1.5; color:var(--text-light); padding-block:.5rem}
.issue-error {color:var(--danger); font-size:.85rem; padding:.6rem 0}
.issue-filter-bar label {display:flex; flex-direction:column; gap:.4rem}
.issue-metrics {display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:.75rem; margin:1.2rem 0}
.issue-metrics div {display:flex; flex-direction:column; gap:.35rem; padding:1rem; background:var(--grey-50); border:1px solid var(--card-border-color); border-radius:10px}
.issue-metrics strong {font-size:1.6rem; line-height:1.1; color:var(--text-strong); font-variant-numeric:tabular-nums}
.issue-metrics span {color:var(--text-light); font-size:.8rem}
.issue-table-wrap {inline-size:100%; max-inline-size:100%; overflow-x:auto; -webkit-overflow-scrolling:touch; border:1px solid var(--card-border-color); border-radius:10px}
.issue-table {inline-size:100%; min-inline-size:830px; border-collapse:collapse; font-size:.87rem}
.issue-table th {text-align:start; white-space:nowrap; background:var(--grey-50); font-size:.78rem; font-weight:750; color:var(--text-strong); padding:.9rem}
.issue-table td {vertical-align:middle; padding:.9rem; border-block-start:1px solid var(--card-border-color)}
.issue-table tbody tr:hover {background:var(--grey-50)}
.issue-title {display:block; font-weight:700; color:var(--text-strong)}
.issue-id {display:block; font-size:.74rem; color:var(--text-light); margin-block-start:.3rem}
.issue-type {display:inline-flex; padding:.25rem .5rem; background:var(--grey-100); border-radius:6px; font-weight:700}
.issue-type.is-bug {background:color-mix(in srgb,var(--danger) 13%,var(--white)); color:var(--danger)}
.issue-empty {text-align:center; color:var(--text-light); padding:2.5rem}
@media (max-width: $tablet) {.issue-header {flex-direction:column; align-items:flex-start}.issue-metrics {grid-template-columns:repeat(2,minmax(0,1fr))}}

/* Unified GitHub Issues visual language: restrained panels and dense issue rows. */
.issue-center {
	--issue-border: var(--card-border-color);
	--issue-muted: var(--grey-50);
}
.issue-header {
	padding-block: .8rem 1rem;
	border-block-end: 1px solid var(--issue-border);
}
.issue-header h1 {
	font-size: clamp(1.45rem, 2.1vw, 1.95rem);
	letter-spacing: -.025em;
}
.issue-header .issue-overline {
	letter-spacing: .08em;
	font-size: .68rem;
}
.issue-filter-bar {
	padding: .85rem;
	background: var(--issue-muted);
	border: 1px solid var(--issue-border);
	border-radius: 8px;
}
.issue-metrics {
	gap: 0;
	border: 1px solid var(--issue-border);
	border-radius: 8px;
	overflow: hidden;
	margin: 1rem 0;
}
.issue-metrics div {
	border: 0;
	border-radius: 0;
	background: var(--white);
	padding: .7rem 1rem;
}
.issue-metrics div + div { border-inline-start: 1px solid var(--issue-border); }
.issue-metrics strong { font-size: 1.25rem; }
.issue-table-wrap { border-radius: 8px; }
.issue-table { font-size: .85rem; }
.issue-table th {
	padding: .7rem .85rem;
	border-bottom: 1px solid var(--issue-border);
	font-weight: 650;
}
.issue-table td { padding: .75rem .85rem; }
.issue-title {
	display: flex;
	align-items: baseline;
	gap: .65rem;
	font-size: .91rem;
}
.issue-title:hover { color: var(--primary); text-decoration: underline; }
.issue-state-mark {
	font-size: 1rem;
	color: #1a7f37;
	flex: 0 0 auto;
}
.issue-state-mark.state-resolved { color: #8250df; }
.issue-state-mark.state-working { color: #bf8700; }
.issue-state-mark.state-review { color: #0969da; }
.issue-id { padding-inline-start: 1.65rem; margin-block-start: .15rem; }
.issue-table select { padding: .4rem .55rem; border-radius: 6px; font-size: .82rem; }
@media (max-width: $tablet) {
	.issue-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
	.issue-metrics div:nth-child(3) { border-inline-start: 0; border-block-start: 1px solid var(--issue-border); }
	.issue-metrics div:nth-child(4) { border-block-start: 1px solid var(--issue-border); }
}

.issue-center>.issue-header {
	background:var(--white);border:1px solid var(--card-border-color);
	border-radius:9px;padding:1rem 1.2rem;margin-block-end:1rem;
}
.issue-header h1 {font-size:clamp(1.3rem,2.4vw,1.75rem);}
.issue-results-toolbar {display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:.65rem;margin-block:.65rem;color:var(--text-light);font-size:.8rem;}
.issue-results-toolbar strong {color:var(--text-strong);font-variant-numeric:tabular-nums;}
.issue-reset-filters {border:1px solid var(--card-border-color);border-radius:6px;background:var(--white);color:var(--text-strong);padding:.38rem .65rem;font-size:.78rem;cursor:pointer;}
.issue-reset-filters:hover {background:var(--grey-50);color:var(--primary);}
.issue-reset-filters:focus-visible {outline:2px solid var(--primary);outline-offset:2px;}
.issue-filter-bar {gap:.7rem;align-items:end;}
.issue-metrics {margin-block:.75rem 1rem;}
.issue-metrics strong {font-variant-numeric:tabular-nums;}
.issue-table-wrap {max-inline-size:100%;}
.issue-table tr:focus-within {background:var(--grey-50);}
.issue-table .issue-title {font-weight:550;line-height:1.45;}
.issue-form label {min-inline-size:0;}
@media (max-width:$tablet) {.issue-center>.issue-header{padding:.8rem;}.issue-filter-bar{padding:.6rem;}.issue-table td{padding:.6rem;}}
</style>
