<template>
	<aside
		:class="{'is-active': baseStore.menuActive, 'is-resizing': isResizing}"
		class="menu-container"
		:style="{'--sidebar-width': sidebarWidthStyle}"
	>
		<nav
			class="menu top-menu"
			:aria-label="$t('navigation.main')"
		>
			<RouterLink
				:to="{name: 'home'}"
				class="logo"
				:aria-label="$t('navigation.home')"
			>
				<Logo
					width="220"
					height="48"
				/>
			</RouterLink>
			<menu class="menu-list other-menu-items">
				<li class="workspace-nav-heading" aria-hidden="true">工作区</li>
				<li>
					<RouterLink
						v-shortcut="SHORTCUTS.navigation.overview"
						:to="{ name: 'home'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="calendar" />
						</span>
						{{ $t('navigation.overview') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="SHORTCUTS.navigation.upcoming"
						:to="{ name: 'tasks.range'}"
					>
						<span class="menu-item-icon icon">
							<Icon :icon="['far', 'calendar-alt']" />
						</span>
						{{ $t('navigation.upcoming') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="SHORTCUTS.navigation.projects"
						:to="{ name: 'projects.index'}"
						
					>
						<span class="menu-item-icon icon">
							<Icon icon="layer-group" />
						</span>
						{{ $t('project.projects') }}
					</RouterLink>
				</li>

				<li>
					<RouterLink
						v-shortcut="SHORTCUTS.navigation.labels"
						:to="{ name: 'labels.index'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="tags" />
						</span>
						{{ $t('label.title') }}
					</RouterLink>
				</li>
				<li>
					<RouterLink
						v-shortcut="SHORTCUTS.navigation.teams"
						:to="{ name: 'teams.index'}"
					>
						<span class="menu-item-icon icon">
							<Icon icon="users" />
						</span>
						{{ $t('team.title') }}
					</RouterLink>
				</li>
				<li class="workspace-nav-heading workspace-nav-divider" aria-hidden="true">团队协作</li>
				<li v-if="authStore.info?.is_admin && !authStore.isLinkShareAuth">
					<RouterLink :to="{name: 'admin.organization'}">
						<span class="menu-item-icon icon"><Icon icon="users" /></span>
						组织任务中心
					</RouterLink>
				</li>
				<li v-if="!authStore.isLinkShareAuth">
					<RouterLink :to="{name: 'issues.index'}">
						<span class="menu-item-icon icon"><svg aria-hidden="true" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M8 7 6 4M16 7l2-3M4 10h3M17 10h3M4 15h3M17 15h3M8 20l-2 2M16 20l2 2"/><rect x="7" y="7" width="10" height="13" rx="5"/><path d="M9 4h6M12 10v8"/></svg></span>
						Bug 管理
					</RouterLink>
				</li>
				<li v-if="!authStore.isLinkShareAuth">
					<RouterLink :to="{name: 'daily-worklogs'}">
						<span class="menu-item-icon icon"><Icon :icon="['far', 'clock']" /></span>
						工时统计
					</RouterLink>
				</li>

			</menu>
		</nav>

		<Loading
			v-if="projectList.isLoading"
			variant="small"
		/>
		<template v-else>
			<nav
				v-if="favoriteProjects.length"
				class="menu"
				:aria-label="$t('project.pseudo.favorites.title')"
			>
				<ProjectsNavigation
					:model-value="favoriteProjects"
					:can-edit-order="false"
					:can-collapse="false"
				/>
			</nav>
			
			<nav
				v-if="savedFilterProjects.length"
				class="menu"
				:aria-label="$t('navigation.savedFilters')"
			>
				<ProjectsNavigation
					:model-value="savedFilterProjects"
					:can-edit-order="false"
					:can-collapse="false"
				/>
			</nav>

			<nav
				class="menu sidebar-projects-section"
				:aria-label="$t('project.projects')"
			>
				<div class="sidebar-projects-heading">项目列表</div>
				<ProjectsNavigation
					:model-value="projects"
					:can-edit-order="true"
					:can-collapse="true"
				/>
			</nav>
		</template>

		<div class="sidebar-app-footer mbs-auto" aria-label="开天眼项目管理">
			<strong>开天眼项目管理</strong>
			<small>内部项目协作平台</small>
		</div>

		<div
			v-if="!isMobile"
			class="resize-handle"
			@mousedown="startResize"
			@touchstart="startResize"
		/>
	</aside>
</template>

<script setup lang="ts">
import {computed} from 'vue'

import {SHORTCUTS} from '@/constants/shortcuts'
import Logo from '@/components/home/Logo.vue'
import Loading from '@/components/misc/Loading.vue'

import {useBaseStore} from '@/stores/base'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import ProjectsNavigation from '@/components/home/ProjectsNavigation.vue'
import {useSidebarResize} from '@/composables/useSidebarResize'

const baseStore = useBaseStore()
const authStore = useAuthStore()
const projectList = useProjects()

const {sidebarWidthStyle, isResizing, startResize, isMobile} = useSidebarResize()

const projects = computed(() => projectList.notArchivedRootProjects)
const favoriteProjects = computed(() => projectList.favoriteProjects)
const savedFilterProjects = computed(() => projectList.savedFilterProjects)
</script>

<style lang="scss" scoped>
.logo {
	display: block;

	padding-inline-start: 1rem;
	margin-inline-end: 1rem;
	margin-block-end: 1rem;

	@media screen and (min-width: $tablet) {
		display: none;
	}
}

.menu-container {
	--sidebar-width: #{$navbar-width};

	display: flex;
	flex-direction: column;
	background: var(--white);
	color: var(--text);
	border-inline-end: 1px solid var(--card-border-color);
	box-shadow: 5px 0 24px rgba(15,23,42,.025);
	padding: 1rem .65rem;
	transition: transform $transition-duration ease-in;
	position: fixed;
	inset-block-start: $navbar-height;
	inset-block-end: 0;
	inset-inline-start: 0;
	transform: translateX(-100%);
	inline-size: var(--sidebar-width);
	overflow-y: auto;

	[dir="rtl"] & {
		transform: translateX(100%);
	}

	@media screen and (max-width: $tablet) {
		inset-block-start: 0;
		inline-size: 70vw;
		z-index: 20;
	}

	&.is-active {
		transform: translateX(0);
		transition: transform $transition-duration ease-out;
	}

	&.is-resizing {
		transition: none;
	}
}

.resize-handle {
	position: absolute;
	inset-block-start: 0;
	inset-block-end: 0;
	inset-inline-end: 0;
	inline-size: 4px;
	cursor: ew-resize;
	background: transparent;
	transition: background-color $transition-duration ease;
	touch-action: none;

	&:hover,
	&:active {
		background-color: var(--primary);
	}
}

.top-menu .menu-list {
	li { font-weight: 620; font-family: $vikunja-font; margin-block: .15rem; }
	.list-menu-link, li > a {
		display: flex;
		align-items: center;
		gap: .7rem;
		min-block-size: 2.65rem;
		padding: .65rem .85rem;
		border-radius: 10px;
		text-decoration: none;
		font-size: .91rem;
		color: var(--text);
		transition: background .15s ease, color .15s ease;
		&:hover { background: var(--grey-50); color: var(--text-strong); }
		&.router-link-active { background: color-mix(in srgb, var(--primary) 11%, var(--white)); color: var(--primary); font-weight: 760; }
		&.router-link-active.nav-muted-active { background: transparent; color: var(--text); font-weight: 620; }
		.icon { min-inline-size: 1.2rem; color: inherit; padding: 0; }
		&:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
	}
}
.menu-container :deep(.navigation-item) {
	margin-inline: .05rem;
	border-radius: 9px;
}
.menu-container :deep(.list-menu-link) {
	border-radius: 9px;
}
.menu-container :deep(.list-menu-link:hover) {
	background-color: var(--grey-50);
}
.menu-container :deep(.project-menu-title) {
	font-size: .87rem;
	font-weight: 550;
}

.menu + .menu {
	padding-block-start: math.div($navbar-padding, 2);
}
.sidebar-projects-section {
	margin-block-start: .75rem;
	padding-block-start: .8rem !important;
	border-block-start: 1px solid var(--card-border-color);
}
.sidebar-projects-heading {
	font-size: .72rem;
	font-weight: 550;
	letter-spacing: .02em;
	color: var(--text-light);
	padding: 0 .85rem .55rem;
}

/* Consistent compact navigation, without changing permission-gated entries. */
.top-menu .menu-list>.workspace-nav-heading {
	padding:.85rem .8rem .3rem;margin:0;
	font-size:.69rem;font-weight:700;letter-spacing:.07em;color:var(--text-light);
	text-transform:uppercase;user-select:none;
}
.top-menu .menu-list>.workspace-nav-divider {margin-block-start:.7rem;border-block-start:1px solid var(--card-border-color);}
.menu-container .top-menu .menu-list>li>a {border-radius:7px;min-block-size:2.45rem;font-size:.86rem;padding:.55rem .75rem;}
.menu-container .top-menu .menu-list>li>a:focus-visible {outline:2px solid var(--primary);outline-offset:1px;}
.menu-container .top-menu .menu-list>li>a.router-link-active {box-shadow:inset 3px 0 0 var(--primary);}
.menu-container .sidebar-projects-section {border-block-start:1px solid var(--card-border-color);padding-block-start:.65rem;}
@media screen and (max-width:$tablet){
	.menu-container {inline-size:min(85vw,340px);max-inline-size:100vw;overscroll-behavior:contain;}
	.top-menu .menu-list>.workspace-nav-heading {padding-block-start:.75rem;}
}
@media(prefers-reduced-motion:reduce){.menu-container{transition:none;}}

.sidebar-app-footer {display:flex;flex-direction:column;gap:.2rem;align-items:center;border-block-start:1px solid var(--card-border-color);margin-block-start:1.2rem;padding:.95rem .3rem .45rem;text-align:center;}
.sidebar-app-footer strong {color:var(--text-light);font-size:.75rem;font-weight:650;}
.sidebar-app-footer small {color:var(--grey-400);font-size:.68rem;}
</style>
