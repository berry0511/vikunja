import {describe, expect, it} from 'vitest'
import {uploadLimitBytes, uploadSizeError} from './uploadSize'

describe('attachment size preflight', () => {
	it('accepts the standard server configuration units', () => {
		expect(uploadLimitBytes('20MB')).toBe(20 * 1024 * 1024)
		expect(uploadLimitBytes('200 MiB')).toBe(200 * 1024 * 1024)
		expect(uploadLimitBytes('1GB')).toBe(1024 ** 3)
		expect(uploadLimitBytes('unlimited')).toBeNull()
	})
	it('blocks an APK larger than the advertised server limit', () => {
		expect(uploadSizeError({size: 60 * 1024 ** 2, name: 'demo.apk'}, '20MB'))
			.toContain('files.maxsize')
		// Increasing the server setting allows this client-side check through.
		expect(uploadSizeError({size: 60 * 1024 ** 2, name: 'demo.apk'}, '200MB')).toBeNull()
	})
	it('enforces different limits for ordinary files and videos', () => {
		expect(uploadSizeError({name: 'archive.zip', size: 1024 ** 3}, '5GB')).toBeNull()
		expect(uploadSizeError({name: 'archive.zip', size: 1024 ** 3 + 1}, '5GB')).toContain('1 GiB')
		expect(uploadSizeError({name: 'demo.MP4', size: 5 * 1024 ** 3}, '5GB')).toBeNull()
		expect(uploadSizeError({name: 'demo.MP4', size: 5 * 1024 ** 3 + 1}, '5GB')).toContain('5 GiB')
		expect(uploadSizeError({name: 'demo.mp4', size: 2 * 1024 ** 3}, '20MB')).toContain('files.maxsize')
	})
	it('allows the exact boundary and leaves unknown configs to the server', () => {
		expect(uploadSizeError({size: 20 * 1024 ** 2, name: 'sample.apk'}, '20MB')).toBeNull()
		expect(uploadSizeError({size: 100, name: 'sample.apk'}, 'unknown')).toBeNull()
	})
})
