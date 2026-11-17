// Normalize rich-text audit snapshots for human-readable, *text-only* display.
// Never render historical user HTML with v-html: it can contain unsafe markup.
export function readableHistoryText(value: string): string {
	if (!value) return ''
	if (!/<\/?(?:p|div|a|br|h[1-6]|blockquote|ul|ol|li|strong|em|span)\b/i.test(value)) {
		return value
	}
	const doc = new DOMParser().parseFromString(value, 'text/html')
	doc.querySelectorAll('script,style,iframe,object,embed,svg').forEach(node => node.remove())
	doc.querySelectorAll('br').forEach(node => node.replaceWith(doc.createTextNode('\n')))
	doc.querySelectorAll('p,div,h1,h2,h3,h4,h5,h6,blockquote,li').forEach(node => {
		node.before(doc.createTextNode('\n'))
		node.after(doc.createTextNode('\n'))
	})
	return (doc.body.textContent ?? '').replace(/\u00a0/g, ' ').replace(/\n{3,}/g, '\n\n').trim()
}

export type HistorySpan = {text: string, changed: boolean}

// A stable Unicode-aware prefix/suffix diff makes a small edit stand out
// without introducing a runtime HTML diff renderer or XSS surface.
export function historyDiffSpans(before: string, after: string, side: 'old' | 'new'): HistorySpan[] {
	const a = Array.from(readableHistoryText(before))
	const b = Array.from(readableHistoryText(after))
	let prefix = 0
	while (prefix < a.length && prefix < b.length && a[prefix] === b[prefix]) prefix++
	let suffix = 0
	while (suffix < a.length - prefix && suffix < b.length - prefix &&
		a[a.length - 1 - suffix] === b[b.length - 1 - suffix]) suffix++
	const selected = side === 'old' ? a : b
	const pieces: HistorySpan[] = [
		{text: selected.slice(0, prefix).join(''), changed: false},
		{text: selected.slice(prefix, selected.length - suffix).join(''), changed: true},
		{text: selected.slice(selected.length - suffix).join(''), changed: false},
	]
	return pieces.filter(p => p.text.length > 0)
}
