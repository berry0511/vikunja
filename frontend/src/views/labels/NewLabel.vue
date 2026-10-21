<template>
	<CreateEdit
		v-if="authStore.info?.is_admin"
		v-model:loading="loadingModel"
		:title="$t('label.create.title')"
		:primary-disabled="label.title === ''"
		@create="newLabel()"
	>
		<FormField
			id="labelTitle"
			v-model="label.title"
			v-focus
			:label="$t('label.attributes.title')"
			:disabled="showLoading"
			:loading="loading"
			:placeholder="$t('label.attributes.titlePlaceholder')"
			type="text"
			:error="showError && label.title === '' ? $t('label.create.titleRequired') : null"
			@keyup.enter="newLabel()"
		/>
		<FormField :label="$t('label.attributes.color')">
			<ColorPicker v-model="label.hex_color" />
		</FormField>
	</CreateEdit>
	<p v-else class="help">标签由实例管理员统一维护。请从现有公共标签中选择。</p>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useAuthStore} from '@/stores/auth'
import {useI18n} from 'vue-i18n'
import {useRouter} from 'vue-router'

import CreateEdit from '@/components/misc/CreateEdit.vue'
import ColorPicker from '@/components/input/ColorPicker.vue'
import FormField from '@/components/input/FormField.vue'

import {useTitle} from '@/composables/useTitle'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {success} from '@/message'
import {getRandomColorHex} from '@/helpers/color/randomColor'
import {createLabelDraft, useCreateLabelMutation} from '@/client/queries/labels'

const router = useRouter()
const authStore = useAuthStore()

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('label.create.title'))

const label = ref(createLabelDraft({hex_color: getRandomColorHex()}))
const createLabelMutation = useCreateLabelMutation()

const showError = ref(false)
const loading = computed(() => createLabelMutation.isPending.value)
const showLoading = useDelayedLoading(loading)
const isSubmitting = ref(false)

const loadingModel = computed({
	get: () => isSubmitting.value || loading.value,
	set(value: boolean) {
		isSubmitting.value = value
	},
})

async function newLabel() {
	if (label.value.title === '') {
		showError.value = true
		return
	}
	showError.value = false

	if (isSubmitting.value) {
		return
	}

	isSubmitting.value = true

	try {
		const newLabel = await createLabelMutation.mutateAsync(label.value)
		router.push({
			name: 'labels.index',
			params: {id: newLabel.id},
		})
		success({message: t('label.create.success')})
	} finally {
		isSubmitting.value = false
	}
}
</script>
