<template>
	<div
		class="card"
		:class="{'has-no-shadow': !shadow}"
	>
		<header
			v-if="title !== ''"
			class="card-header"
		>
			<p class="card-header-title">
				{{ title }}
			</p>
			<BaseButton
				v-if="showClose"
				class="card-header-icon close"
				:aria-label="$t('misc.close')"
				@click="$emit('close')"
			>	
				<span class="icon">
					<Icon icon="times" />
				</span>
			</BaseButton>
		</header>
		<div
			class="card-content loader-container"
			:class="{
				'p-0': !padding,
				'is-loading': showLoading
			}"
		>
			<div :class="{'content': hasContent}">
				<slot />
			</div>
		</div>

		<footer
			v-if="$slots.footer"
			class="card-footer"
		>
			<slot name="footer" />
		</footer>
	</div>
</template>

<script setup lang="ts">
import BaseButton from '@/components/base/BaseButton.vue'
import {useDelayedLoading} from '@/composables/useDelayedLoading'

const props = withDefaults(defineProps<{
	title?: string
	padding?: boolean
	shadow?: boolean
	hasContent?: boolean
	loading?: boolean
	showClose?: boolean
}>(), {
	title: '',
	padding: true,
	shadow: true,
	hasContent: true,
	loading: false,
	showClose: false,
})

defineEmits<{
	'close': []
}>()

const showLoading = useDelayedLoading(() => props.loading)
</script>

<style lang="scss" scoped>
.card {
	max-inline-size: 100%;
	position: relative;
	margin-block-end: 1.15rem;
	background-color: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 14px;
	box-shadow: 0 4px 18px rgba(15,23,42,.045);
	color: var(--text);
	// Keep overflow visible: dropdowns and date selectors inside report cards
	// must not be clipped by rounded panel boundaries.
	&.has-no-shadow { box-shadow: none; }
	@media print { box-shadow: none; border: 0; }
}
.card-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	background: transparent;
	box-shadow: none;
	border-block-end: 1px solid var(--card-border-color);
}
.card-header-title {
	flex-grow: 1;
	color: var(--text-strong);
	font-size: .96rem;
	font-weight: 750;
	line-height: 1.4;
	padding: 1rem 1.25rem;
	&.is-centered { justify-content: center; }
}
.card-header-icon {
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	padding: .75rem 1rem;
}
.card-content {
	background: transparent;
	padding: clamp(1rem, 2vw, 1.35rem);
	&.p-0 { padding: 0; }
}
.card-footer {
	display: flex;
	justify-content: flex-end;
	align-items: center;
	background: var(--grey-50);
	padding: 1rem 1.25rem;
	border-block-start: 1px solid var(--card-border-color);
}
</style>
