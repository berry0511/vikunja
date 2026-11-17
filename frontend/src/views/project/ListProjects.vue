<template>
	<section v-cy="'projects-list'" class="project-directory loader-container" :class="{'is-loading': loading}">
		<header class="directory-header">
			<div class="directory-intro">
				<p class="directory-eyebrow">{{ $t('project.projects') }} / WORKSPACE</p>
				<h1>{{ $t('project.projects') }}</h1>
				<p class="directory-description">{{ $t('project.directory.subtitle') }}</p>
			</div>
			<div class="directory-actions">
				<XButton :to="{name: 'filters.create'}" variant="secondary" icon="filter">{{ $t('filters.create.title') }}</XButton>
				<XButton v-cy="'new-project'" :to="{name: 'project.create'}" icon="plus">{{ $t('project.create.header') }}</XButton>
			</div>
		</header>
		<div class="directory-toolbar">
			<div class="directory-tabs" role="group" :aria-label="$t('project.directory.filterLabel')">
				<button type="button" :class="{'is-active': status === 'active'}" :aria-pressed="status === 'active'" @click="setStatus('active')">
					{{ $t('project.directory.active') }} <span>{{ activeProjects.length }}</span>
				</button>
				<button v-cy="'show-archived-check'" type="button" :class="{'is-active': status === 'archived'}" :aria-pressed="status === 'archived'" @click="setStatus('archived')">
					<Icon icon="archive" /> {{ $t('project.directory.archived') }} <span>{{ archivedProjects.length }}</span>
				</button>
				<button type="button" :class="{'is-active': status === 'all'}" :aria-pressed="status === 'all'" @click="setStatus('all')">
					{{ $t('project.directory.all') }} <span>{{ allProjects.length }}</span>
				</button>
			</div>
			<label class="directory-search">
				<Icon icon="search" />
				<input v-model.trim="searchText" type="search" aria-label="搜索当前项目列表" placeholder="搜索项目名称…" />
			</label>
			<p class="directory-view-hint">{{ status === 'archived' ? $t('project.directory.archiveHint') : $t('project.directory.openHint') }}</p>
		</div>
		<div v-if="projects.length === 0 && !loading" class="directory-empty">
			<Icon :icon="status === 'archived' ? 'archive' : 'layer-group'" />
			<h2>{{ status === 'archived' ? $t('project.directory.emptyArchived') : $t('project.directory.empty') }}</h2>
			<p>{{ status === 'archived' ? $t('project.directory.emptyArchivedHint') : $t('project.directory.emptyHint') }}</p>
		</div>
		<ProjectCardGrid v-else :projects="projects" show-archived />
	</section>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import ProjectCardGrid from '@/components/project/partials/ProjectCardGrid.vue'
import {useTitle} from '@/composables/useTitle'
import {useProjects} from '@/composables/useProjects'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {useI18n} from 'vue-i18n'

const {t} = useI18n()
useTitle(() => t('project.title'))
const route = useRoute()
const router = useRouter()
const projectList = useProjects()
const loading = useDelayedLoading(() => projectList.isLoading)
const allProjects = computed(() => projectList.projectsArray.filter(p => p.id > 0))
const activeProjects = computed(() => allProjects.value.filter(p => !p.is_archived))
const archivedProjects = computed(() => allProjects.value.filter(p => p.is_archived))
const searchText = ref('')
type ProjectStatus = 'active' | 'archived' | 'all'
const status = computed<ProjectStatus>(() =>
	route.query.status === 'archived' || route.query.status === 'all' ? route.query.status : 'active',
)
const projects = computed(() => {
	const source = status.value === 'archived'
		? archivedProjects.value
		: status.value === 'all' ? allProjects.value : activeProjects.value
	const term = searchText.value.toLocaleLowerCase()
	return term ? source.filter(project => project.title.toLocaleLowerCase().includes(term)) : source
})
function setStatus(next: ProjectStatus) {
	if (next === status.value) return
	void router.replace({name: 'projects.index', query: {...route.query, status: next === 'active' ? undefined : next}})
}
</script>

<style lang="scss" scoped>
.project-directory {
	max-inline-size: 1440px;
	margin-inline: auto;
	padding: clamp(.5rem, 1.7vw, 1.5rem);
	min-block-size: 45vh;
}
.directory-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1.5rem;
	margin-block-end: 1.5rem;
}
.directory-intro { min-inline-size: 0; }
.directory-eyebrow {
	font-size: .73rem;
	font-weight: 800;
	letter-spacing: .12em;
	text-transform: uppercase;
	color: var(--primary);
	margin-block-end: .55rem;
}
.directory-header h1 {
	font-size: clamp(1.6rem, 3vw, 2.3rem);
	line-height: 1.25;
	font-weight: 750;
	color: var(--text-strong);
	margin: 0 0 .45rem;
}
.directory-description, .directory-view-hint { color: var(--text-light); font-size: .9rem; }
.directory-actions { display: flex; flex-wrap: wrap; gap: .65rem; flex-shrink: 0; }
.directory-toolbar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	padding: .8rem;
	margin-block-end: 1.4rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 14px;
	box-shadow: 0 3px 14px rgba(15,23,42,.035);
}
.directory-tabs {
	display: flex;
	flex-wrap: wrap;
	gap: .3rem;
	padding: .25rem;
	background: var(--grey-50);
	border-radius: 10px;
}
.directory-tabs button {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: .5rem;
	font: inherit;
	font-size: .88rem;
	font-weight: 650;
	border: 0;
	border-radius: 8px;
	background: transparent;
	color: var(--text-light);
	cursor: pointer;
	min-block-size: 2.35rem;
	padding: .45rem .85rem;
	transition: background .15s ease, box-shadow .15s ease, color .15s ease;
}
.directory-tabs button:hover { color: var(--text-strong); background: var(--grey-100); }
.directory-tabs button.is-active {
	color: var(--text-strong);
	background: var(--white);
	box-shadow: 0 2px 8px rgba(15,23,42,.09);
}
.directory-tabs button:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
.directory-tabs button span { opacity: .65; font-size: .77rem; }
.directory-view-hint { margin-inline-end: .6rem; }
.directory-empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	min-block-size: 270px;
	text-align: center;
	gap: .65rem;
	border: 1px dashed var(--card-border-color);
	border-radius: 16px;
	background: var(--white);
	color: var(--text-light);
}
.directory-empty > svg { font-size: 1.6rem; color: var(--primary); }
.directory-empty h2 { font-size: 1.1rem; font-weight: 700; color: var(--text-strong); margin: 0; }
@media (max-width: $tablet) {
	.directory-header, .directory-toolbar { align-items: flex-start; flex-direction: column; }
	.directory-actions { inline-size: 100%; }
	.directory-view-hint { padding-inline: .35rem; }
}

.directory-header {padding:1.1rem 1.2rem;background:var(--white);border:1px solid var(--card-border-color);border-radius:9px;margin-block-end:.85rem;}
.directory-header h1 {font-size:clamp(1.35rem,2.3vw,1.85rem);}
.directory-toolbar {border-radius:8px;gap:.75rem;margin-block-end:1rem;}
.directory-search {display:flex;align-items:center;gap:.5rem;min-inline-size:180px;max-inline-size:300px;flex:1;background:var(--white);border:1px solid var(--card-border-color);border-radius:7px;padding:.42rem .7rem;color:var(--text-light);}
.directory-search input {border:0;background:transparent;min-inline-size:0;inline-size:100%;font:inherit;font-size:.84rem;color:var(--text);outline:none;}
.directory-search:focus-within {outline:2px solid var(--primary);outline-offset:1px;}
.directory-tabs {border:1px solid var(--card-border-color);}
.directory-tabs button.is-active {color:var(--primary);}
.directory-view-hint {max-inline-size:260px;font-size:.75rem;}
@media(max-width:$tablet){.directory-header{padding:.85rem;}.directory-toolbar{align-items:stretch;}.directory-search{min-inline-size:100%;max-inline-size:100%;}.directory-view-hint{max-inline-size:none;}}
</style>
