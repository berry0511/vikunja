import type {Task} from '@/client/generated'

// Bug reports reuse the existing task ACL, discussion, attachments and worklogs.
// Work-order classification is deliberately retired: ordinary work is a task.
export const ISSUE_LABELS = {
	bug: 'Bug',
} as const

export type IssueKind = 'bug'
export type IssueStatus = 'open' | 'working' | 'review' | 'resolved'
type TicketTask = Pick<Task, 'done' | 'percent_done' | 'labels'>

export function issueKind(task: TicketTask): IssueKind | null {
	const titles = (task.labels ?? []).map(label => label.title)
	if (titles.some(title => [ISSUE_LABELS.bug, '工单·缺陷', '工单-缺陷'].includes(title))) return 'bug'
	return null
}

export function issueStatus(task: Pick<Task, 'done' | 'percent_done'>): IssueStatus {
	if (task.done) return 'resolved'
	if ((task.percent_done ?? 0) >= .9) return 'review'
	if ((task.percent_done ?? 0) > 0) return 'working'
	return 'open'
}

export function statusTaskPatch(status: IssueStatus): Pick<Task, 'done' | 'percent_done'> {
	switch (status) {
		case 'working': return {done: false, percent_done: .5}
		case 'review': return {done: false, percent_done: .9}
		case 'resolved': return {done: true, percent_done: 1}
		default: return {done: false, percent_done: 0}
	}
}

export const ISSUE_STATUS_LABELS: Record<IssueStatus, string> = {
	open: '待处理',
	working: '处理中',
	review: '待验收',
	resolved: '已完成',
}

// Rich-text descriptions must not interpret untrusted user-provided text as
// markup. Issue fields remain editable in the normal task detail editor.
function escapeHTML(text: string): string {
	return text.replace(/&/g, '&amp;').replace(/</g, '&lt;')
		.replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

export function issueDescription(input: {
	kind: IssueKind
	details: string
	environment: string
	steps: string
	expected: string
	actual: string
}): string {
	const fields: Array<[string, string]> = [
		['环境 / 版本', input.environment], ['复现步骤', input.steps],
		['预期结果', input.expected], ['实际结果', input.actual],
		['补充说明', input.details],
	]
	return fields.filter(([, value]) => value.trim()).map(([title, value]) =>
		`<h3>${escapeHTML(title)}</h3><p>${escapeHTML(value).replace(/\n/g, '<br>')}</p>`,
	).join('') || '<p></p>'
}
