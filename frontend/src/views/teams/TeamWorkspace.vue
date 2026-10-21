<template>
	<div class="team-workspace">
		<header class="team-workspace-heading">
			<div>
				<RouterLink :to="{name: 'teams.index'}">← 团队列表</RouterLink>
				<h1>{{ team?.name ?? '团队工作区' }}</h1>
				<p>{{ team?.description || '团队成员与协作空间' }}</p>
			</div>
			<RouterLink class="team-workspace-manage" :to="{name: 'teams.edit', params: {id: teamId}}">管理团队</RouterLink>
		</header>
		<p v-if="teamQuery.isPending.value">正在加载团队…</p>
		<p v-else-if="teamQuery.isError.value" role="alert">无法读取团队，请检查团队成员权限。</p>
		<section v-else class="team-workspace-panel">
			<h2>团队成员 <small>({{ team?.members?.length ?? 0 }})</small></h2>
			<ul class="team-member-list">
				<li v-for="member in team?.members ?? []" :key="member.id">
					<span>{{ member.name || member.username }}</span>
					<span class="team-member-role">{{ member.admin ? '团队管理员' : '成员' }}</span>
				</li>
			</ul>
			<div class="team-project-help">
				<h2>团队任务</h2>
				<p>团队任务属于共享给该团队的项目，不包括成员私人任务。需要先在项目的共享设置中为团队授权，才能按团队查看和编辑。</p>
				<RouterLink :to="{name: 'projects.index'}">查看有权限的项目 →</RouterLink>
			</div>
		</section>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {teamQuery as getTeamQuery} from '@/client/queries/teams'

const props = defineProps<{id: string}>()
const teamId = computed(() => Number(props.id))
const teamQuery = useQuery(() => getTeamQuery(teamId.value))
const team = computed(() => teamQuery.data.value)
</script>

<style scoped lang="scss">
.team-workspace {max-inline-size: 1100px; margin-inline: auto; padding: 1.2rem;}
.team-workspace-heading {display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:1rem; padding:1rem 1.2rem; margin-block-end:1rem; background:var(--white); border:1px solid var(--card-border-color); border-radius:10px;}
.team-workspace-heading h1 {font-size:1.45rem; font-weight:600; margin:.5rem 0 .2rem;}
.team-workspace-heading p {font-size:.85rem; color:var(--text-light);}
.team-workspace-manage {border:1px solid var(--card-border-color); border-radius:6px; padding:.45rem .75rem;}
.team-workspace-panel {border:1px solid var(--card-border-color); border-radius:10px; padding:1rem 1.2rem; background:var(--white);}
.team-workspace-panel h2 {font-size:1rem; font-weight:600;}
.team-workspace-panel small {color:var(--text-light); font-weight:400;}
.team-member-list {list-style:none; margin: .75rem 0 1.5rem; padding:0;}
.team-member-list li {display:flex; justify-content:space-between; gap:1rem; padding:.65rem 0; border-bottom:1px solid var(--card-border-color);}
.team-member-role {color:var(--text-light); font-size:.8rem;}
.team-project-help {padding-block-start:.75rem;}
.team-project-help p {font-size:.85rem; margin:.7rem 0; color:var(--text-light);}
</style>
