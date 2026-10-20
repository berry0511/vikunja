<template>
	<section class="entity-history">
		<header class="history-header">
			<div>
				<h2>修改历史</h2>
				<p>系统记录实际字段变更；普通成员无法编辑或删除历史记录。</p>
			</div>
			<XButton type="button" variant="secondary" :icon="expanded ? 'chevron-up' : 'history'" @click="expanded = !expanded">
				{{ expanded ? '收起历史' : '查看历史' }}
			</XButton>
		</header>
		<div v-if="expanded" class="history-body">
			<p v-if="history.isPending.value" class="history-message">正在加载修改记录…</p>
			<p v-else-if="history.isError.value" class="history-error" role="alert">无法读取修改历史：{{ getErrorText(history.error.value) }}</p>
			<p v-else-if="!history.data.value?.items.length" class="history-message">暂无字段修改记录。</p>
			<ol v-else class="history-list">
				<li v-for="entry in history.data.value.items" :key="entry.id" class="history-entry">
					<div class="history-meta">
						<span class="history-actor">{{ entry.username || '#' + entry.actor_id }}</span>
						<span class="history-field">{{ entry.action === 'delete' ? '删除' : (fieldNames[entry.field] || entry.field) }}</span>
						<time :datetime="entry.created">{{ formatDate(entry.created) }}</time>
					</div>
					<div v-if="entry.action !== 'delete'" class="history-values">
						<div><span>修改前</span><pre>{{ entry.old_value || '（空）' }}</pre></div>
						<div><span>修改后</span><pre>{{ entry.new_value || '（空）' }}</pre></div>
					</div>
					<p v-else class="history-message">该对象被删除，历史记录仍保留。</p>
				</li>
			</ol>
			<div v-if="(history.data.value?.total_pages ?? 0) > 1" class="history-pagination">
				<XButton variant="secondary" :disabled="page <= 1" @click="page--">上一页</XButton>
				<span>{{ page }} / {{ history.data.value?.total_pages }}</span>
				<XButton variant="secondary" :disabled="page >= (history.data.value?.total_pages ?? 0)" @click="page++">下一页</XButton>
			</div>
		</div>
	</section>
</template>

<script setup lang="ts">
import {ref, watch} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {changeHistoryQuery} from '@/client/queries/changeHistory'
import {getErrorText} from '@/message'

const props = defineProps<{kind: 'task' | 'project', entityId: number}>()
const expanded = ref(false)
const page = ref(1)
watch(() => props.entityId, () => {page.value = 1; expanded.value = false})
const history = useQuery(() => ({
	...changeHistoryQuery(props.kind, props.entityId, page.value),
	enabled: expanded.value && props.entityId > 0,
}))

const fieldNames: Record<string, string> = {
	title: '标题', description: '详细说明', start_date: '开始日期',
	end_date: '结束日期', due_date: '截止时间', done_at: '完成时间',
	project_id: '所属项目', done: '完成状态', priority: '优先级',
	percent_done: '任务进度', repeat_after: '重复周期', repeat_mode: '重复模式',
	hex_color: '颜色', bucket_id: '任务分组', cover_image_attachment_id: '封面附件',
	is_archived: '归档状态', identifier: '项目标识', parent_project_id: '父项目',
	position: '排序位置', deleted: '删除操作',
}
function formatDate(input: string): string {
	const date = new Date(input)
	return Number.isNaN(date.getTime()) ? input : date.toLocaleString()
}
</script>

<style scoped lang="scss">
.entity-history {
	margin-block: 1.2rem;
	padding: 1rem;
	border-radius: 12px;
	border: 1px solid var(--card-border-color);
	background: var(--white);
}
.history-header { display: flex; justify-content: space-between; align-items: center; gap: 1rem; }
.history-header h2 { font-size: 1rem; font-weight: 750; color: var(--text-strong); margin: 0 0 .3rem; }
.history-header p, .history-message { font-size: .82rem; line-height: 1.5; color: var(--text-light); }
.history-body { margin-block-start: 1rem; }
.history-list { list-style: none; margin: 0; padding: 0; display: grid; gap: .8rem; }
.history-entry { padding: .9rem; border-radius: 10px; border: 1px solid var(--card-border-color); background: var(--grey-50); }
.history-meta { display: flex; flex-wrap: wrap; align-items: center; gap: .65rem; font-size: .82rem; margin-block-end: .6rem; }
.history-actor { font-weight: 750; color: var(--text-strong); }
.history-field { padding: .2rem .5rem; background: var(--white); color: var(--primary); border-radius: 6px; font-weight: 700; }
.history-meta time { color: var(--text-light); margin-inline-start: auto; font-variant-numeric: tabular-nums; }
.history-values { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: .75rem; }
.history-values > div { min-inline-size: 0; }
.history-values span { color: var(--text-light); font-size: .75rem; font-weight: 700; }
.history-values pre {
	white-space: pre-wrap;
	word-break: break-word;
	max-block-size: 220px;
	overflow: auto;
	padding: .6rem;
	margin: .3rem 0 0;
	border-radius: 7px;
	font-family: inherit;
	font-size: .84rem;
	line-height: 1.45;
	color: var(--text);
	background: var(--white);
	border: 1px solid var(--card-border-color);
}
.history-error { color: var(--danger); }
.history-pagination { display: flex; align-items: center; justify-content: center; gap: .8rem; padding-block-start: 1rem; }
@media (max-width: $tablet) {
	.history-header { align-items: flex-start; flex-direction: column; }
	.history-values { grid-template-columns: 1fr; }
}
</style>
