import {queryOptions} from '@tanstack/vue-query'
import {client} from '@/client/generated/client.gen'

export type ChangeHistory = {
	id: number
	entity_type: 'task' | 'project'
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

export function changeHistoryQuery(kind: 'task' | 'project', id: number, page = 1) {
	return queryOptions({
		queryKey: ['change-history', kind, id, page],
		queryFn: async () => {
			const {data} = await client.get<ChangeHistoryPage>({
				url: kind === 'task' ? '/tasks/{id}/history' : '/projects/{id}/history',
				path: {id},
				query: {page},
			})
			return data
		},
		enabled: id > 0,
		staleTime: 0,
	})
}
