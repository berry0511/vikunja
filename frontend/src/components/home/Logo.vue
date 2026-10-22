<script setup lang="ts">
import {computed} from 'vue'
import {useColorScheme} from '@/composables/useColorScheme'

const {isDark} = useColorScheme()
const customLogo = computed(() => {
	const light = window.CUSTOM_LOGO_URL
	const dark = window.CUSTOM_LOGO_URL_DARK
	return (isDark.value ? dark || light : light || dark) || '/images/kty-logo.svg'
})
</script>

<template>
	<div class="kty-brand">
		<img :src="customLogo" class="kty-brand-mark" alt="" aria-hidden="true">
		<span class="kty-brand-name">开天眼项目管理</span>
	</div>
</template>

<style scoped lang="scss">
.kty-brand {
	display: inline-flex;
	align-items: center;
	justify-content: flex-start;
	gap: .6rem;
	inline-size: max-content;
	max-inline-size: 100%;
	min-block-size: 45px;
}
.kty-brand-mark {
	inline-size: 42px;
	block-size: 42px;
	flex: none;
	object-fit: contain;
}
.kty-brand-name {
	font-size: 1.04rem;
	font-weight: 780;
	letter-spacing: .012em;
	color: var(--text-strong);
	white-space: nowrap;
}
@media screen and (max-width: $tablet) {
	.kty-brand-mark {inline-size: 36px; block-size: 36px;}
	.kty-brand-name {font-size: .95rem;}
}
</style>
