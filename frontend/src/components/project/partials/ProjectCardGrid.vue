<template>
	<ul
		class="project-grid"
		:class="{ 'show-even-number-of-projects': showEvenNumberOfProjects }"
	>
		<li
			v-for="(item, index) in filteredProjects"
			:key="`project_${item.id}_${index}`"
			class="project-grid-item"
		>
			<ProjectCard :project="item" />
		</li>
	</ul>
</template>

<script lang="ts" setup>
import {computed} from 'vue'
import type {ProjectResponse} from '@/client/queries/projects'

import ProjectCard from './ProjectCard.vue'

const props = withDefaults(defineProps<{
	projects: ProjectResponse[],
	showArchived?: boolean,
	itemLimit?: boolean,
	showEvenNumberOfProjects?: boolean,
}>(), {
	showArchived: false,
	itemLimit: false,
	showEvenNumberOfProjects: false,
})

const filteredProjects = computed(() => {
	return props.showArchived
		? props.projects
		: props.projects.filter(project => !project.is_archived)
})
</script>

<style lang="scss" scoped>
.project-grid {
	margin: 0;
	list-style: none;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr));
	align-items: stretch;
	gap: 1.1rem;
}
.project-grid-item {
	margin: 0 !important;
	min-inline-size: 0;
	display: block;
}
.project-grid-item :deep(.project-card) { block-size: 100%; }
.show-even-number-of-projects {
	@media screen and (min-width: $widescreen) {
		.project-grid-item:nth-child(5) { display: none; }
	}
}
</style>
