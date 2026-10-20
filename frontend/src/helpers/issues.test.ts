import {describe, expect, it} from 'vitest'
import {ISSUE_LABELS, issueKind, issueStatus, statusTaskPatch, issueDescription} from './issues'

describe('issue workflow mapped onto existing tasks', () => {
	it('classifies by reserved labels without confusing normal tasks', () => {
		expect(issueKind({labels: []})).toBeNull()
		expect(issueKind({labels: [{title: ISSUE_LABELS.bug}]})).toBe('bug')
		expect(issueKind({labels: [{title: ISSUE_LABELS.work}]})).toBe('work')
	})
	it('maps workflow stages into persisted task completion and progress', () => {
		for (const status of ['open', 'working', 'review', 'resolved'] as const) {
			expect(issueStatus(statusTaskPatch(status))).toBe(status)
		}
	})
	it('escapes descriptions to prevent field data from becoming HTML', () => {
		const html = issueDescription({
			kind: 'bug', details: '', environment: '<script>alert(1)</script>',
			steps: 'one\ntwo', expected: 'works', actual: 'broken',
		})
		expect(html).not.toContain('<script>')
		expect(html).toContain('&lt;script&gt;')
		expect(html).toContain('one<br>two')
	})
})
