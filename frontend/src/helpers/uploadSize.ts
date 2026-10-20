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

const VIDEO_EXTENSION = /\.(mp4|m4v|mov|avi|mkv|webm|wmv|flv|mpeg|mpg|3gp|ts)$/i
const GIB = 1024 ** 3

export function uploadSizeError(file: Pick<File, 'size' | 'name'>, maxSize: string): string | null {
	const isVideo = VIDEO_EXTENSION.test(file.name)
	const desired = isVideo ? 5 * GIB : GIB
	const server = uploadLimitBytes(maxSize)
	const limit = server === null ? desired : Math.min(server, desired)
	if (file.size <= limit) return null
	return `文件「${file.name}」超过允许上传大小（${isVideo ? '视频 5 GiB' : '普通文件 1 GiB'}）。当前服务配置上限为 ${maxSize}，请同时检查 files.maxsize 和反向代理限制。`
}
