// Preserve the generated API client's authentication and response handling while
// exposing actual upload bytes for large multipart attachments.
export function uploadProgressFetch(onProgress: (loaded: number, total: number) => void): typeof fetch {
	return async (request) => {
		const prepared = request instanceof Request ? request : new Request(request)
		const form = await prepared.formData()
		return await new Promise<Response>((resolve, reject) => {
			const xhr = new XMLHttpRequest()
			xhr.open(prepared.method, prepared.url)
			prepared.headers.forEach((value, key) => {
				// XHR generates a new multipart boundary for FormData.
				if (key.toLowerCase() !== 'content-type') xhr.setRequestHeader(key, value)
			})
			xhr.upload.onprogress = event => {
				if (event.lengthComputable && event.total > 0) onProgress(event.loaded, event.total)
			}
			xhr.onload = () => {
				const headers = new Headers()
				for (const line of xhr.getAllResponseHeaders().trim().split(/[\r\n]+/)) {
					const split = line.indexOf(':')
					if (split > 0) headers.append(line.slice(0, split).trim(), line.slice(split + 1).trim())
				}
				resolve(new Response(xhr.status === 204 ? null : xhr.responseText, {status: xhr.status, headers}))
			}
			xhr.onerror = () => reject(new TypeError('Attachment upload network error'))
			xhr.ontimeout = () => reject(new TypeError('Attachment upload timed out'))
			xhr.onabort = () => reject(new DOMException('Upload aborted', 'AbortError'))
			if (prepared.signal.aborted) { reject(new DOMException('Upload aborted', 'AbortError')); return }
			prepared.signal.addEventListener('abort', () => xhr.abort(), {once: true})
			xhr.send(form)
		})
	}
}
