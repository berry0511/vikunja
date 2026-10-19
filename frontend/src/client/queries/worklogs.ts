import {queryOptions, useMutation, useQuery} from '@tanstack/vue-query'
import {client} from '@/client/generated/client.gen'
import {contextMutationOptions} from './contextMutation'
import {expectBlob} from './blobResponse'
import type {MaybeRefOrGetter} from 'vue'
import {toValue} from 'vue'

export type Worklog = {
	id: number
	user_id: number
	username: string
	project_id: number
	task_id: number
	work_date: string
	minutes: number
	comment: string
	created: string
	updated: string
}

export type WorklogWrite = Pick<Worklog, 'project_id' | 'task_id' | 'work_date' | 'minutes' | 'comment'>

export type WorklogFilter = {
	project_id: number
	user_id: number
	from_date: string
	to_date: string
}

export type WorklogPage = {
	items: Worklog[]
	total: number
	page: number
	per_page: number
	total_pages: number
}

export type WorklogSummary = {
	days: Array<{work_date: string, user_id: number, username: string, minutes: number}>
	total_minutes: number
	record_count: number
}

const keys = {
	all: ['daily-worklogs'] as const,
	list: (filter: WorklogFilter, page: number) => ['daily-worklogs', 'list', filter, page] as const,
	summary: (filter: WorklogFilter) => ['daily-worklogs', 'summary', filter] as const,
}

export function worklogListQuery(filter: WorklogFilter, page: number) {
	return queryOptions({
		queryKey: keys.list(filter, page),
		enabled: filter.project_id > 0,
		queryFn: async ({signal}): Promise<WorklogPage> => {
			const {data} = await client.get<{200: WorklogPage}, unknown, true>({
				url: '/worklogs',
				query: {...filter, page},
				signal,
				throwOnError: true,
			})
			return data
		},
	})
}

export function worklogSummaryQuery(filter: WorklogFilter) {
	return queryOptions({
		queryKey: keys.summary(filter),
		enabled: filter.project_id > 0,
		queryFn: async ({signal}): Promise<WorklogSummary> => {
			const {data} = await client.get<{200: WorklogSummary}, unknown, true>({
				url: '/worklogs/summary',
				query: filter,
				signal,
				throwOnError: true,
			})
			return data
		},
	})
}

export function useWorklogs(filter: MaybeRefOrGetter<WorklogFilter>, page: MaybeRefOrGetter<number>) {
	return useQuery(() => worklogListQuery(toValue(filter), toValue(page)))
}

export function useWorklogSummary(filter: MaybeRefOrGetter<WorklogFilter>) {
	return useQuery(() => worklogSummaryQuery(toValue(filter)))
}

export function useCreateWorklog() {
	return useMutation(contextMutationOptions({
		mutationFn: async (body: WorklogWrite) => {
			const {data} = await client.post<{201: Worklog}, unknown, true>({
				url: '/worklogs', body, throwOnError: true,
			})
			return data
		},
		onSettled: (_input, queryClient) => queryClient.invalidateQueries({queryKey: keys.all}),
	}))
}

export function useUpdateWorklog() {
	return useMutation(contextMutationOptions({
		mutationFn: async ({id, ...body}: WorklogWrite & {id: number}) => {
			const {data} = await client.put<{200: Worklog}, unknown, true>({
				url: '/worklogs/{id}', path: {id}, body, throwOnError: true,
			})
			return data
		},
		onSettled: (_input, queryClient) => queryClient.invalidateQueries({queryKey: keys.all}),
	}))
}

export function useDeleteWorklog() {
	return useMutation(contextMutationOptions({
		mutationFn: async (id: number) => {
			await client.delete<{204: unknown}, unknown, true>({
				url: '/worklogs/{id}', path: {id}, throwOnError: true,
			})
		},
		onSettled: (_input, queryClient) => queryClient.invalidateQueries({queryKey: keys.all}),
	}))
}

// Uses the same generated HTTP transport/interceptors as other endpoints.
// Regenerate the SDK from the new Go OpenAPI schema when the full build
// toolchain is available; no generated file is modified by this patch.
export async function downloadWorklogs(filter: WorklogFilter): Promise<Blob> {
	const {data} = await client.get<{200: Blob}, unknown, true>({
		url: '/worklogs/export',
		query: filter,
		parseAs: 'blob',
		throwOnError: true,
	})
	return expectBlob(data, 'daily worklogs CSV')
}

export async function downloadWorklogSummary(filter: WorklogFilter): Promise<Blob> {
	const {data} = await client.get<{200: Blob}, unknown, true>({
		url: '/worklogs/export-summary',
		query: filter,
		parseAs: 'blob',
		throwOnError: true,
	})
	return expectBlob(data, 'daily worklogs summary CSV')
}
