<template>
	<section class="entity-history" :class="{'history-compact': compact}">
		<header v-if="!compact" class="history-header">
			<div>
				<h2>{{ kind === 'comment' ? '评论修改记录' : '修改历史' }}</h2>
				<p>系统记录实际字段变更；普通成员无法编辑或删除历史记录。</p>
			</div>
			<XButton type="button" variant="secondary" :icon="expanded ? 'chevron-up' : 'history'" @click="expanded = !expanded">
				{{ expanded ? '收起历史' : '查看历史' }}
			</XButton>
		</header>
		<button v-if="compact" type="button" class="history-text-trigger" :aria-expanded="expanded" @click="expanded = !expanded">
			修改记录
		</button>
		<div v-if="expanded" class="history-body">
			<div v-if="compact" class="history-panel-top">
				<strong>评论修改历史</strong>
				<button type="button" class="history-close" @click="expanded = false">关闭</button>
			</div>
			<p v-if="history.isPending.value" class="history-message">正在加载修改记录…</p>
			<p v-else-if="history.isError.value" class="history-error" role="alert">无法读取修改历史：{{ getErrorText(history.error.value) }}</p>
			<p v-else-if="!history.data.value?.items.length" class="history-message">暂无字段修改记录。</p>
			<ol v-else class="history-list">
				<li v-for="entry in history.data.value.items" :key="entry.id" class="history-entry">
					<div class="history-meta">
						<span class="history-actor">{{ entry.username || '#' + entry.actor_id }}</span>
						<span class="history-field">{{ entry.field === 'attachment' && entry.action === 'upload' ? '上传附件' : entry.field === 'attachment' && entry.action === 'delete' ? '删除附件' : entry.action === 'delete' ? '删除' : (fieldNames[entry.field] || entry.field) }}</span>
						<time :datetime="entry.created">{{ formatDate(entry.created) }}</time>
					</div>
					<div v-if="entry.action !== 'delete' || entry.field === 'attachment'" class="history-values">
						<div class="history-before">
							<span class="history-value-label">修改前</span>
							<pre><template v-if="entry.old_value"><span v-for="(part, i) in historyDiffSpans(entry.old_value, entry.new_value, 'old')" :key="i" :class="{'history-removed': part.changed}">{{ part.text }}</span></template><span v-else>（空）</span></pre>
						</div>
						<div class="history-after">
							<span class="history-value-label">修改后</span>
							<pre><template v-if="entry.new_value"><span v-for="(part, i) in historyDiffSpans(entry.old_value, entry.new_value, 'new')" :key="i" :class="{'history-added': part.changed}">{{ part.text }}</span></template><span v-else>（空）</span></pre>
						</div>
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
import {historyDiffSpans} from '@/helpers/historyDiff'

const props = withDefaults(defineProps<{kind: 'task' | 'project' | 'comment', entityId: number, taskId?: number, compact?: boolean}>(), {compact: false})
const expanded = ref(false)
const page = ref(1)
watch(() => [props.entityId, props.taskId], () => {page.value = 1; expanded.value = false})
const history = useQuery(() => ({
	...changeHistoryQuery(props.kind, props.entityId, page.value, props.taskId ?? 0),
	enabled: expanded.value && props.entityId > 0 && (props.kind !== 'comment' || (props.taskId ?? 0) > 0),
}))

const fieldNames: Record<string, string> = {
	title: '标题', description: '详细说明', start_date: '开始日期',
	end_date: '结束日期', due_date: '截止时间', done_at: '完成时间',
	project_id: '所属项目', done: '完成状态', priority: '优先级',
	percent_done: '任务进度', repeat_after: '重复周期', repeat_mode: '重复模式',
	hex_color: '颜色', bucket_id: '任务分组', cover_image_attachment_id: '封面附件',
	is_archived: '归档状态', identifier: '项目标识', parent_project_id: '父项目',
	position: '排序位置', assignee: '负责人', label: '标签', attachment: '附件', comment: '评论正文', deleted: '删除操作',
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
.history-compact {
	position: relative;
	margin: 0 0 0 auto;
	padding: 0;
	border: 0;
	background: transparent;
	flex-shrink: 0;
}
.history-text-trigger {
	border: 0;
	background: transparent;
	color: var(--primary);
	padding: .3rem .4rem;
	border-radius: 6px;
	font-size: .8rem;
	font-weight: 740;
	cursor: pointer;
	white-space: nowrap;
	&:hover { background: var(--grey-50); text-decoration: underline; }
	&:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
}
.history-compact .history-body {
	position: absolute;
	z-index: 50;
	top: calc(100% + .45rem);
	right: 0;
	inline-size: min(650px, calc(100vw - 3rem));
	max-block-size: min(70vh, 640px);
	overflow-y: auto;
	background: var(--white);
	padding: 1rem;
	border-radius: 12px;
	border: 1px solid var(--card-border-color);
	box-shadow: 0 16px 38px rgba(15,23,42,.18);
}
.history-panel-top {display: flex; align-items: center; justify-content: space-between; gap: .8rem; padding-bottom: .75rem;}
.history-panel-top strong {font-size: .94rem; color: var(--text-strong);}
.history-close {background: transparent; border: 0; color: var(--primary); font-weight: 700; cursor: pointer;}
.history-value-label {
	display: inline-block;
	padding: .25rem .65rem;
	margin-bottom: .45rem;
	border-radius: 6px;
	font-size: .8rem !important;
	font-weight: 750 !important;
}
.history-before .history-value-label {background: var(--grey-100); color: var(--text-strong);}
.history-after .history-value-label {background: color-mix(in srgb, var(--primary) 15%, var(--white)); color: var(--primary);}
.history-before pre {border-color: var(--grey-300); background: var(--grey-50);}
.history-after pre {border-color: color-mix(in srgb, var(--primary) 26%, var(--card-border-color)); background: color-mix(in srgb, var(--primary) 5%, var(--white));}
.history-removed {background: color-mix(in srgb, var(--danger) 19%, var(--white)); color: var(--text-strong); text-decoration: line-through; border-radius: 3px;}
.history-added {background: color-mix(in srgb, var(--success) 25%, var(--white)); color: var(--text-strong); font-weight: 720; border-radius: 3px;}
@media (max-width: $tablet) {
	.history-compact .history-body {right: -1rem; inline-size: min(650px, calc(100vw - 2rem));}
}
.history-error { color: var(--danger); }
.history-pagination { display: flex; align-items: center; justify-content: center; gap: .8rem; padding-block-start: 1rem; }
@media (max-width: $tablet) {
	.history-header { align-items: flex-start; flex-direction: column; }
	.history-values { grid-template-columns: 1fr; }
}
</style>
