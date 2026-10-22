import {describe, expect, it} from 'vitest'
import {historyDiffSpans, readableHistoryText} from './historyDiff'

describe('change history readable comparison', () => {
	it('converts rich comment HTML to readable text without exposing markup', () => {
		const original = '<p><a href="/api/v2/tasks/1/attachments/4">MILESTONE.zip</a><br>123</p>'
		const cleaned = readableHistoryText(original)
		expect(cleaned).toContain('MILESTONE.zip')
		expect(cleaned).toContain('123')
		expect(cleaned).not.toContain('<a')
		expect(cleaned).not.toContain('href=')
	})
	it('highlights only the updated Chinese characters', () => {
		const old = historyDiffSpans('工单待处理', '工单已处理', 'old')
		const next = historyDiffSpans('工单待处理', '工单已处理', 'new')
		expect(old.filter(x => x.changed).map(x => x.text)).toEqual(['待'])
		expect(next.filter(x => x.changed).map(x => x.text)).toEqual(['已'])
	})
	it('renders added text without a fake old value', () => {
		expect(historyDiffSpans('', '新增说明', 'old')).toEqual([])
		expect(historyDiffSpans('', '新增说明', 'new')).toEqual([{text: '新增说明', changed: true}])
	})
	it('renders untouched values without change highlighting', () => {
		expect(historyDiffSpans('unchanged', 'unchanged', 'new')).toEqual([{text: 'unchanged', changed: false}])
	})
})
