<template>
	<div
		class="project-card"
		:class="{
			'has-light-text': Boolean(background),
			'has-background': Boolean(blurHashUrl || background)
		}"
		:style="{
			'border-inline-start': project.hex_color ? `0.25rem solid ${project.hex_color}` : undefined,
			'background-image': blurHashUrl ? `url(${blurHashUrl})` : undefined,
		}"
	>
		<div
			class="project-background background-fade-in"
			:class="{'is-visible': background}"
			:style="{'background-image': background ? `url(${background})` : undefined}"
		/>
		<span v-if="teamShared" class="shared-team-badge"><Icon icon="users" /> 团队共享</span>
		<span
			v-if="project.is_archived"
			class="is-archived"
		>{{ $t('project.archived') }}</span>

		<div class="project-card-heading" aria-hidden="true">
			<span class="project-card-kicker"><Icon :icon="project.id < -1 ? 'filter' : 'folder'" /> {{ project.id < -1 ? $t('filters.title') : $t('project.projects') }}</span>
			<div class="project-title">{{ getProjectTitle(project) }}</div>
			<p v-if="textOnlyDescription" class="project-card-description">{{ textOnlyDescription }}</p>
			<span class="project-card-enter">进入项目 <span aria-hidden="true">→</span></span>
		</div>
		<BaseButton
			class="project-button"
			:aria-label="getProjectTitle(project)"
			:title="textOnlyDescription"
			:to="{
				name: 'project.index',
				params: { projectId: project.id}
			}"
		/>
		<BaseButton
			v-if="!project.is_archived && project.id > -1"
			class="favorite"
			:aria-label="project.is_favorite ? $t('project.unfavorite') : $t('project.favorite')"
			:class="{'is-favorite': project.is_favorite}"
			@click.prevent.stop="toggleProjectFavorite"
		>
			<Icon :icon="project.is_favorite ? 'star' : ['far', 'star']" />
		</BaseButton>
	</div>
</template>

<script lang="ts" setup>
import {computed} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {projectTeamSharesQuery} from '@/client/queries/projectShares'
import {usePatchProjectFavoriteMutation, type ProjectResponse} from '@/client/queries/projects'

import BaseButton from '@/components/base/BaseButton.vue'

import {useProjectBackground} from '@/composables/useProjectBackground'
import {getProjectTitle} from '@/helpers/getProjectTitle'

const props = defineProps<{
	project: ProjectResponse,
}>()

const teamShares = useQuery(computed(() => ({
	...projectTeamSharesQuery(props.project.id),
	enabled: props.project.id > 0,
	retry: false,
})))
const teamShared = computed(() => (teamShares.data.value?.length ?? 0) > 0)

const {background, blurHashUrl} = useProjectBackground(() => props.project)

const favoriteMutation = usePatchProjectFavoriteMutation()

async function toggleProjectFavorite() {
	if (props.project.id <= 0 || props.project.is_archived) {
		return
	}
	await favoriteMutation.mutateAsync({id: props.project.id, isFavorite: !props.project.is_favorite})
}

const textOnlyDescription = computed(() => {
	return props.project.description ? props.project.description.replace(/<[^>]*>/g, '') : ''
})
</script>

<style lang="scss" scoped>
.project-card {
	position: relative;
	display: flex;
	flex-direction: column;
	justify-content: center;
	min-block-size: 132px;
	padding: 1rem 1.15rem;
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	background: var(--white);
	box-shadow: 0 2px 7px rgba(15,23,42,.035);
	overflow: hidden;
	transition: border-color .15s ease, box-shadow .15s ease, transform .15s ease;
}
.project-card:hover {border-color: color-mix(in srgb,var(--primary) 35%,var(--card-border-color));box-shadow:0 7px 18px rgba(15,23,42,.07);transform:translateY(-1px);}
.project-card:focus-within {outline:2px solid var(--primary);outline-offset:2px;}
.project-card.has-background {background-size:cover;background-position:center;min-block-size:156px;}
.project-background,.project-button {position:absolute;inset:0;}
.project-background {background-size:cover;background-position:center;background-repeat:no-repeat;}
.project-card-heading {position:relative;z-index:1;display:flex;flex-direction:column;gap:.4rem;padding-inline-end:2rem;pointer-events:none;}
.project-card-kicker {display:inline-flex;align-items:center;gap:.4rem;font-size:.72rem;font-weight:500;color:var(--text-light);}
.project-title {color:var(--text-strong);font-size:1.05rem;font-weight:600;line-height:1.4;overflow-wrap:anywhere;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;}
.project-card-description {margin:0;color:var(--text-light);font-size:.79rem;line-height:1.5;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden;}
.has-background .project-card-heading {justify-content:flex-end;text-shadow:0 1px 7px rgba(0,0,0,.65);}
.has-background .project-title,.has-background .project-card-kicker,.has-background .project-card-description {color:#fff;}
.project-button {z-index:2;}
.favorite {position:absolute!important;z-index:4;top:.7rem;right:.7rem;background:var(--white);border-radius:8px;width:2rem;height:2rem;display:flex;align-items:center;justify-content:center;color:var(--text-light);}
.favorite:hover,.favorite.is-favorite {color:var(--warning);}
.shared-team-badge,.is-archived {position:relative;z-index:3;align-self:flex-start;display:inline-flex;align-items:center;gap:.35rem;font-size:.7rem;font-weight:550;border-radius:6px;padding:.2rem .5rem;margin-bottom:.55rem;}
.shared-team-badge {background:color-mix(in srgb,var(--primary) 12%,var(--white));color:var(--primary);}
.is-archived {background:var(--grey-100);color:var(--text);}
.background-fade-in {opacity:0;transition:opacity .2s ease;}
.background-fade-in.is-visible {opacity:1;}

.project-card{min-block-size:150px;justify-content:flex-start;}
.project-card-heading{flex:1;gap:.5rem;}
.project-card-enter{font-size:.74rem;color:var(--primary);font-weight:650;margin-block-start:auto;padding-block-start:.7rem;}
.has-background .project-card-enter{color:#fff;}
.project-title{font-weight:650;}
@media(max-width:720px){.project-card{min-block-size:126px;padding:.85rem;}.project-card-heading{gap:.35rem;}}
@media(prefers-reduced-motion:reduce){.project-card{transition:none;transform:none!important;}}
</style>
