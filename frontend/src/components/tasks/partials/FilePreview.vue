<template>
	<!-- Preview image -->
	<img
		v-if="blobUrl"
		:src="blobUrl"
		alt="Attachment preview"
	>

	<!-- PDF icon -->
	<div
		v-else-if="isPdf"
		class="icon-wrapper"
	>
		<Icon
			size="lg"
			icon="file-pdf"
		/>
	</div>

	<!-- Audio icon -->
	<div
		v-else-if="isAudio"
		class="icon-wrapper"
	>
		<Icon
			size="lg"
			icon="volume-high"
		/>
	</div>

	<!-- Video icon -->
	<div
		v-else-if="isVideo"
		class="icon-wrapper"
	>
		<Icon
			size="lg"
			icon="play"
		/>
	</div>

	<!-- Fallback -->
	<div
		v-else
		class="icon-wrapper"
	>
		<Icon
			size="lg"
			icon="file"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {fetchAttachmentUrl, releaseAttachmentUrl} from '@/helpers/attachments'
import type {TaskAttachment as IAttachment} from '@/client/generated'
import type {AttachmentIdentity} from '@/client/queries/attachments'
import {canPreviewAudio, canPreviewImage, canPreviewPdf, canPreviewVideo} from '@/helpers/attachmentPreview'

const props = defineProps<{
	modelValue?: IAttachment
}>()

const blobUrl = ref<string | undefined>(undefined)
const isPdf = computed(() => props.modelValue && canPreviewPdf(props.modelValue))
const isAudio = computed(() => props.modelValue && canPreviewAudio(props.modelValue))
const isVideo = computed(() => props.modelValue && canPreviewVideo(props.modelValue))

function isPreviewable(attachment?: IAttachment): attachment is IAttachment & AttachmentIdentity {
	return Boolean(attachment?.id && attachment.task_id && canPreviewImage(attachment))
}

// Keyed on the ids, not the prop object: a list refetch hands over an equal attachment as a new object.
watch(
	() => isPreviewable(props.modelValue) ? `${props.modelValue.task_id}-${props.modelValue.id}` : null,
	async (key, _previous, onCleanup) => {
		releaseAttachmentUrl(blobUrl.value)
		blobUrl.value = undefined
		const attachment = props.modelValue
		if (key === null || !isPreviewable(attachment)) {
			return
		}

		let stale = false
		onCleanup(() => {
			stale = true
		})

		try {
			const url = await fetchAttachmentUrl(attachment, 'md')
			if (stale) {
				releaseAttachmentUrl(url)
				return
			}
			blobUrl.value = url
		} catch {
			// keep the generic file icon
		}
	},
	{immediate: true},
)

onBeforeUnmount(() => releaseAttachmentUrl(blobUrl.value))
</script>

<style scoped lang="scss">
img {
	inline-size: 100%;
	block-size: 100%;
	max-inline-size: 100%;
	max-block-size: 100%;
	border-radius: $radius;
	object-fit: contain;
}

.icon-wrapper {
	display: flex;
	align-items: center;
	justify-content: center;
	inline-size: 100%;
	block-size: 100%;
	font-size: .95rem;
	overflow: hidden;
	:deep(svg) {max-inline-size: 1.65rem; max-block-size: 1.65rem;}
	color: var(--grey-500);
}
</style>
