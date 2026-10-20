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
		<span
			v-if="project.is_archived"
			class="is-archived"
		>{{ $t('project.archived') }}</span>

		<div class="project-card-heading" aria-hidden="true">
			<span class="project-card-kicker"><Icon :icon="project.id < -1 ? 'filter' : 'folder'" /> {{ project.id < -1 ? $t('filters.title') : $t('project.projects') }}</span>
			<div class="project-title">{{ getProjectTitle(project) }}</div>
			<p v-if="textOnlyDescription" class="project-card-description">{{ textOnlyDescription }}</p>
		</div>
		<BaseButton
			class="project-button"
			:aria-label="project.title"
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
import {usePatchProjectFavoriteMutation, type ProjectResponse} from '@/client/queries/projects'

import BaseButton from '@/components/base/BaseButton.vue'

import {useProjectBackground} from '@/composables/useProjectBackground'
import {getProjectTitle} from '@/helpers/getProjectTitle'

const props = defineProps<{
	project: ProjectResponse,
}>()

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
	--project-card-padding: 1.1rem;
	position: relative;
	display: flex;
	align-items: stretch;
	min-block-size: 132px;
	padding: var(--project-card-padding);
	border: 1px solid var(--card-border-color);
	border-radius: 14px;
	background: var(--white);
	box-shadow: 0 3px 13px rgba(15, 23, 42, .035);
	overflow: hidden;
	transition: box-shadow .18s ease, transform .18s ease, border-color .18s ease;

	&.has-background { min-block-size: 155px; background-size: cover; background-position: center; }
	&:hover {
		box-shadow: 0 10px 24px rgba(15, 23, 42, .095);
		border-color: color-mix(in srgb, var(--primary) 30%, var(--card-border-color));
		transform: translateY(-2px);
	}
	&:focus-within { outline: 2px solid var(--primary); outline-offset: 2px; }
	> * { position: relative; }
}
.project-background,
.project-button {
	position: absolute;
	inset: 0;
}
.project-background {
	background-size: cover;
	background-repeat: no-repeat;
	background-position: center;
}
.project-card-heading {
	inline-size: 100%;
	display: flex;
	flex-direction: column;
	justify-content: center;
	gap: .45rem;
	padding-inline-end: 1.2rem;
	z-index: 1;
	pointer-events: none;
}
.project-card-kicker {
	font-size: .69rem;
	font-weight: 800;
	letter-spacing: .075em;
	text-transform: uppercase;
	color: var(--text-light);
	display: inline-flex;
	align-items: center;
	gap: .45rem;
}
.project-title {
	font-family: $vikunja-font;
	font-size: 1.17rem;
	font-weight: 750;
	line-height: 1.32;
	color: var(--text-strong);
	overflow-wrap: anywhere;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	overflow: hidden;
}
.project-card-description {
	font-size: .83rem;
	line-height: 1.45;
	color: var(--text-light);
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	overflow: hidden;
	margin: 0;
}
.has-background .project-card-heading {
	justify-content: flex-end;
	text-shadow: 0 1px 8px rgba(0,0,0,.7);
}
.has-background .project-card-kicker,
.has-background .project-title,
.has-background .project-card-description { color: #fff; }
.project-button { z-index: 2; }
.favorite {
	position: absolute !important;
	z-index: 3;
	inset-block-start: .7rem;
	inset-inline-end: .7rem;
	background: var(--white);
	border-radius: 9px;
	inline-size: 2rem;
	block-size: 2rem;
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--text-light);
	transition: background .15s ease, color .15s ease;
	&:hover, &.is-favorite { color: var(--warning); background: var(--grey-50); }
}
.is-archived {
	position: absolute !important;
	inset-inline-start: 1rem;
	inset-block-start: .75rem;
	font-size: .72rem;
	font-weight: 750;
	padding: .25rem .55rem;
	border-radius: 6px;
	background: var(--grey-100);
	color: var(--text-strong);
	z-index: 1;
}
.project-card:has(.is-archived) .project-card-heading { padding-block-start: 1.45rem; }
.background-fade-in { opacity: 0; transition: opacity $transition; &.is-visible { opacity: 1; } }
</style>
