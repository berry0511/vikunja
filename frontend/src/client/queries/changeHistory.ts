import {queryOptions} from '@tanstack/vue-query'
import {client} from '@/client/generated/client.gen'

export type ChangeHistory = {
	id: number
	entity_type: 'task' | 'project' | 'comment'
	entity_id: number
	actor_id: number
	username: string
	action: string
	field: string
	old_value: string
	new_value: string
	created: string
}
export type ChangeHistoryPage = {
	items: ChangeHistory[]
	total: number
	page: number
	per_page: number
	total_pages: number
}

export function changeHistoryQuery(kind: 'task' | 'project' | 'comment', id: number, page = 1, taskId = 0) {
	return queryOptions({
		queryKey: ['change-history', kind, id, taskId, page],
		queryFn: async () => {
			const {data} = await client.get<{200: ChangeHistoryPage}, unknown, true>({
				url: kind === 'comment'
					? '/tasks/{task}/comments/{id}/history'
					: kind === 'task' ? '/tasks/{id}/history' : '/projects/{id}/history',
				path: kind === 'comment' ? {task: taskId, id} : {id},
				query: {page},
				throwOnError: true,
			})
			return data
		},
		enabled: id > 0 && (kind !== 'comment' || taskId > 0),
		staleTime: 0,
	})
}
