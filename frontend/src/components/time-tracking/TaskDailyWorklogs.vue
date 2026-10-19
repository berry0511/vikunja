<template>
	<section class="task-worklogs">
		<div class="task-worklogs__heading">
			<div>
				<h2 class="task-section-title">每日工时</h2>
				<p class="task-worklogs__hint">工时自动关联到当前任务，无需填写项目或任务编号。</p>
			</div>
			<XButton v-if="!showForm" icon="plus" variant="secondary" @click="openNew">记工时</XButton>
		</div>

		<Card v-if="showForm" :title="editingId ? '修改工时' : '记录今天的工作'">
			<form @submit.prevent="save">
				<div class="task-worklogs__inputs">
					<div class="field">
						<label class="label" for="task-worklog-date">工作日期</label>
						<FormInput id="task-worklog-date" v-model="workDate" type="date" required />
					</div>
					<div class="field">
						<label class="label" for="task-worklog-hours">投入小时</label>
						<FormInput id="task-worklog-hours" v-model.number="hours" type="number" inputmode="decimal" min="0.0167" max="24" step="any" required />
					</div>
				</div>
				<div class="field">
					<label class="label" for="task-worklog-note">工作说明（可选）</label>
					<textarea id="task-worklog-note" v-model="comment" class="textarea task-worklogs__textarea" rows="3" maxlength="2000" placeholder="记录本次工作内容" />
				</div>
				<p v-if="errorText" class="help is-danger" role="alert">{{ errorText }}</p>
				<div class="task-worklogs__actions">
					<XButton type="submit" :loading="saving" :disabled="saving">保存工时</XButton>
					<XButton type="button" variant="secondary" @click="closeForm">取消</XButton>
				</div>
			</form>
		</Card>

		<div class="task-worklogs__summary">
			<div>
				<span class="task-worklogs__small">近一年累计</span>
				<strong>{{ formatMinutes(summaryQuery.data.value?.total_minutes ?? 0) }}</strong>
			</div>
			<span class="task-worklogs__small">{{ summaryQuery.data.value?.record_count ?? 0 }} 条记录</span>
		</div>
		<p v-if="listQuery.isPending.value" class="task-worklogs__hint">正在加载工时记录…</p>
		<div v-else-if="entries.length" class="task-worklogs__entries">
			<div v-for="entry in entries" :key="entry.id" class="task-worklogs__entry">
				<div class="task-worklogs__entry-info">
					<strong>{{ formatMinutes(entry.minutes) }}</strong>
					<span>{{ entry.work_date.slice(0, 10) }} · {{ entry.username || '#' + entry.user_id }}</span>
					<p v-if="entry.comment">{{ entry.comment }}</p>
				</div>
				<div v-if="entry.user_id === currentUserId" class="task-worklogs__entry-actions">
					<XButton type="button" variant="tertiary" @click="edit(entry)">编辑</XButton>
					<XButton type="button" variant="tertiary" :disabled="deleting" @click="remove(entry)">删除</XButton>
				</div>
			</div>
		</div>
		<p v-else class="task-worklogs__hint">这个任务还没有工时记录。</p>
		<RouterLink :to="{name: 'daily-worklogs', query: {project_id: projectId, task_id: taskId}}" class="task-worklogs__more">
			查看工时统计与导出
		</RouterLink>
	</section>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import Card from '@/components/misc/Card.vue'
import FormInput from '@/components/input/FormInput.vue'
import {useAuthStore} from '@/stores/auth'
import {
	useWorklogs,
	useWorklogSummary,
	useCreateWorklog,
	useUpdateWorklog,
	useDeleteWorklog,
	type Worklog,
} from '@/client/queries/worklogs'

const props = defineProps<{taskId: number, projectId: number}>()
const auth = useAuthStore()
const currentUserId = computed(() => auth.info?.id ?? 0)

function localDate(value: Date): string {
	return [value.getFullYear(), String(value.getMonth() + 1).padStart(2, '0'), String(value.getDate()).padStart(2, '0')].join('-')
}
const today = localDate(new Date())
const from = new Date()
from.setFullYear(from.getFullYear() - 1)
const filter = computed(() => ({
	project_id: props.projectId,
	task_id: props.taskId,
	user_id: 0,
	from_date: localDate(from),
	to_date: today,
}))
const listQuery = useWorklogs(filter, () => 1)
const summaryQuery = useWorklogSummary(filter)
const entries = computed(() => listQuery.data.value?.items ?? [])
const create = useCreateWorklog()
const update = useUpdateWorklog()
const del = useDeleteWorklog()
const showForm = ref(false)
const editingId = ref(0)
const workDate = ref(today)
const hours = ref(1)
const comment = ref('')
const errorText = ref('')
const saving = ref(false)
const deleting = ref(false)

function formatMinutes(minutes: number): string {
	return (minutes / 60).toFixed(2) + ' 小时'
}
function closeForm() {
	showForm.value = false
	editingId.value = 0
	workDate.value = today
	hours.value = 1
	comment.value = ''
	errorText.value = ''
}
function openNew() {
	closeForm()
	showForm.value = true
}
function edit(entry: Worklog) {
	editingId.value = entry.id
	workDate.value = entry.work_date.slice(0, 10)
	hours.value = entry.minutes / 60
	comment.value = entry.comment
	errorText.value = ''
	showForm.value = true
}
async function save() {
	const minutes = Math.round(Number(hours.value) * 60)
	if (!workDate.value || !Number.isFinite(minutes) || minutes < 1 || minutes > 1440) {
		errorText.value = '请输入有效日期和 1 分钟～24 小时的工作时长。'
		return
	}
	const body = {
		project_id: props.projectId,
		task_id: props.taskId,
		work_date: workDate.value + 'T00:00:00Z',
		minutes,
		comment: comment.value,
	}
	errorText.value = ''
	saving.value = true
	try {
		if (editingId.value) await update.mutateAsync({id: editingId.value, ...body})
		else await create.mutateAsync(body)
		closeForm()
	} catch {
		errorText.value = '保存失败，请确认你仍有权限访问这个任务。'
	} finally {
		saving.value = false
	}
}
async function remove(entry: Worklog) {
	if (!confirm('确定删除这条工时吗？')) return
	deleting.value = true
	try {
		await del.mutateAsync(entry.id)
	} catch {
		errorText.value = '删除失败，请重试。'
	} finally {
		deleting.value = false
	}
}
watch(() => props.taskId, closeForm)
</script>

<style scoped lang="scss">
.task-worklogs {
	padding-block: 1rem;
	border-block-start: 1px solid var(--card-border-color);
}
.task-worklogs__heading,
.task-worklogs__summary,
.task-worklogs__entry,
.task-worklogs__actions {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
}
.task-worklogs__heading {
	margin-block-end: 1rem;
}
.task-worklogs__hint,
.task-worklogs__small {
	color: var(--text-light);
	font-size: .85rem;
}
.task-worklogs__inputs {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
	gap: 1rem;
}
.task-worklogs__textarea {
	inline-size: 100%;
	min-block-size: 5rem;
}
.task-worklogs__actions {justify-content: flex-start; flex-wrap: wrap;}
.task-worklogs__summary {
	background: var(--grey-50);
	padding: 1rem;
	border-radius: $radius;
	margin-block: 1rem;
}
.task-worklogs__summary strong {
	margin-inline-start: .75rem;
	font-size: 1.35rem;
	color: var(--text-strong);
}
.task-worklogs__entries {border-block-start: 1px solid var(--card-border-color);}
.task-worklogs__entry {
	padding: .8rem 0;
	border-block-end: 1px solid var(--card-border-color);
}
.task-worklogs__entry-info {
	min-inline-size: 0;
	overflow-wrap: anywhere;
}
.task-worklogs__entry-info span {
	margin-inline-start: .75rem;
	color: var(--text-light);
	font-size: .85rem;
}
.task-worklogs__entry-info p {margin-block-start: .3rem;}
.task-worklogs__entry-actions {display: flex; flex-shrink: 0;}
.task-worklogs__more {display: inline-block; margin-block-start: 1rem;}
@media (max-width: 600px) {
	.task-worklogs__heading,
	.task-worklogs__entry {align-items: flex-start; flex-direction: column;}
	.task-worklogs__summary {align-items: flex-start; flex-direction: column; gap: .3rem;}
}
</style>
