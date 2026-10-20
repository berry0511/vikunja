// The server exposes files.maxsize as a human-readable size (typically "20MB").
// Keep a conservative client-side preflight so a rejected large upload is not
// reported as the browser's opaque "Failed to fetch" network error.
export function uploadLimitBytes(raw: string): number | null {
	const match = /^\s*(\d+(?:\.\d+)?)\s*(B|KIB|KB|MIB|MB|GIB|GB|TIB|TB)\s*$/i.exec(raw)
	if (!match) return null
	const value = Number(match[1])
	if (!Number.isFinite(value) || value < 0) return null
	const unit = match[2].toUpperCase()
	const factors: Record<string, number> = {
		B: 1, KB: 1024, KIB: 1024, MB: 1024 ** 2,
		MIB: 1024 ** 2, GB: 1024 ** 3, GIB: 1024 ** 3,
		TB: 1024 ** 4, TIB: 1024 ** 4,
	}
	const bytes = value * factors[unit]
	return Number.isSafeInteger(Math.floor(bytes)) ? Math.floor(bytes) : null
}

export function uploadSizeError(file: Pick<File, 'size' | 'name'>, maxSize: string): string | null {
	const limit = uploadLimitBytes(maxSize)
	if (limit === null || file.size <= limit) return null
	return `文件「${file.name}」超过单文件上传上限 ${maxSize}。请联系管理员调整 files.maxsize 并检查反向代理上传大小限制。`
}
