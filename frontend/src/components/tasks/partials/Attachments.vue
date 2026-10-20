<template>
	<div class="attachments">
		<h2 class="task-section-title">
			<span class="icon is-grey">
				<Icon icon="paperclip" />
			</span>
			{{ $t('task.attachment.title') }}
		</h2>

		<input
			v-if="editEnabled"
			id="files"
			ref="filesRef"
			multiple
			type="file"
			@change="uploadNewAttachment()"
		>

		<div v-if="uploadBatch !== null" class="attachment-upload-progress" role="status">
			<span>正在上传 {{ uploadBatch.completed + 1 }} / {{ uploadBatch.total }}：{{ uploadProgress === undefined ? "准备中…" : Math.round(uploadProgress) + "%" }}</span>
		<ProgressBar
			:value="uploadProgress"
			:aria-label="$t('task.attachment.upload')"
			is-primary
		/>
		</div>

		<div
			v-if="attachments.length > 0"
			class="files"
		>
			<div
				v-for="a in attachments"
				:key="a.id"
				class="attachment"
			>
				<div class="preview-column">
					<button
						class="preview-open"
						tabindex="-1"
						aria-hidden="true"
						@click="viewOrDownload(a)"
					>
						<FilePreview
							class="attachment-preview"
							:model-value="a"
						/>
					</button>
				</div>
				<div class="attachment-info-column">
					<button
						v-tooltip="attachmentMetaTooltip(a)"
						class="attachment-open"
						@click="viewOrDownload(a)"
					>
						<span class="filename">
							{{ a.file?.name }}
							<span
								v-if="task.cover_image_attachment_id === a.id"
								class="is-task-cover"
							>
								{{ $t('task.attachment.usedAsCover') }}
							</span>
						</span>
					</button>
					<p class="attachment-info-meta">
						<User
							:user="a.created_by ?? {}"
							:avatar-size="20"
							:show-username="false"
							:is-inline="true"
						/>
						<span class="attachment-uploader">上传人：{{ getDisplayName(a.created_by ?? {}) }}</span>
						<span v-if="a.created" class="attachment-created">上传时间：{{ formatDateLong(a.created) }}</span>
						<span class="attachment-size">{{ getHumanSize(a.file?.size ?? 0) }}</span>
					</p>
					<AudioPreview
						v-if="canPreviewAudio(a)"
						:ref="el => setAudioPlayerRef(a, el)"
						:attachment="a"
					/>

				</div>
				<div class="attachment-row-actions">
					<p class="attachment-actions">
						<BaseButton
							v-tooltip="$t('task.attachment.downloadTooltip')"
							:aria-label="$t('task.attachment.downloadTooltip')"
							class="attachment-info-meta-button"
							@click.prevent.stop="download(a)"
						>
							<Icon icon="download" />
						</BaseButton>
						<BaseButton
							v-tooltip="$t('task.attachment.copyUrlTooltip')"
							:aria-label="$t('task.attachment.copyUrlTooltip')"
							class="attachment-info-meta-button"
							@click.stop="copyUrl(a)"
						>
							<Icon icon="copy" />
						</BaseButton>
						<BaseButton
							v-if="editEnabled"
							v-tooltip="$t('task.attachment.deleteTooltip')"
							:aria-label="$t('task.attachment.deleteTooltip')"
							class="attachment-info-meta-button attachment-delete-action"
							@click.prevent.stop="setAttachmentToDelete(a)"
						>
							<Icon icon="trash-alt" />
						</BaseButton>
						<BaseButton
							v-if="editEnabled && canPreviewImage(a)"
							v-tooltip="task.cover_image_attachment_id === a.id
								? $t('task.attachment.unsetAsCover')
								: $t('task.attachment.setAsCover')"
							:aria-label="task.cover_image_attachment_id === a.id
								? $t('task.attachment.unsetAsCover')
								: $t('task.attachment.setAsCover')"
							class="attachment-info-meta-button"
							@click.prevent.stop="setCoverImage(task.cover_image_attachment_id === a.id ? null : a)"
						>
							<Icon :icon="task.cover_image_attachment_id === a.id ? 'eye-slash' : 'eye'" />
						</BaseButton>
					</p>
				</div>
			</div>
		</div>

		<XButton
			v-if="editEnabled"
			:aria-disabled="loading || undefined"
			class="mbe-4"
			icon="cloud-upload-alt"
			variant="secondary"
			:shadow="false"
			@click="openFilePicker()"
		>
			{{ $t('task.attachment.upload') }}
		</XButton>

		<!-- Dropzone -->
		<Teleport :to="dropzoneTeleportTarget">
			<div
				v-if="editEnabled"
				:class="{hidden: !showDropzone}"
				class="dropzone"
			>
				<div class="drop-hint">
					<div class="icon">
						<Icon icon="cloud-upload-alt" />
					</div>
					<div class="hint">
						{{ $t('task.attachment.drop') }}
					</div>
				</div>
			</div>
		</Teleport>

		<!-- Delete modal -->
		<Modal
			:enabled="attachmentToDelete !== null"
			@close="setAttachmentToDelete(null)"
			@submit="deleteAttachment()"
		>
			<template #header>
				<span>{{ $t('task.attachment.delete') }}</span>
			</template>

			<template #text>
				<p v-if="attachmentToDelete">
					{{ $t('task.attachment.deleteText1', {filename: attachmentToDelete.file?.name}) }}<br>
					<strong class="has-text-white">{{ $t('misc.cannotBeUndone') }}</strong>
				</p>
			</template>
		</Modal>

		<ImageLightbox
			v-if="preview?.kind === 'image'"
			:key="preview.blobUrl"
			:blob-url="preview.blobUrl"
			:alt="preview.name"
			@close="closePreview"
		/>

		<!-- Attachment PDF modal -->
		<Modal
			:enabled="preview?.kind === 'pdf'"
			:wide="true"
			:aria-label="$t('misc.pdfPreview')"
			@close="closePreview"
		>
			<iframe
				v-if="preview?.kind === 'pdf'"
				:src="preview.blobUrl"
				:title="preview.name"
				class="pdf-preview-iframe"
			/>
		</Modal>

		<!-- Attachment video modal -->
		<Modal
			:enabled="previewLoading || preview?.kind === 'video'"
			:wide="true"
			:aria-label="$t('misc.videoPreview')"
			@close="closePreview"
		>
			<Loading v-if="previewLoading" />
			<div
				v-else-if="previewFailed"
				class="video-preview-error"
			>
				<p>{{ $t('misc.videoLoadFailed') }}</p>
				<XButton
					icon="download"
					variant="secondary"
					@click="downloadVideoPreview"
				>
					{{ $t('misc.download') }}
				</XButton>
			</div>
			<video
				v-else-if="preview?.kind === 'video'"
				:src="preview.blobUrl"
				:aria-label="preview.name"
				class="video-preview"
				controls
				playsinline
				@error="onVideoError"
			/>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, nextTick, watch, onMounted, onBeforeUnmount, type ComponentPublicInstance} from 'vue'
import {useDropZone} from '@vueuse/core'

import User from '@/components/misc/User.vue'
import ProgressBar from '@/components/misc/ProgressBar.vue'
import Loading from '@/components/misc/Loading.vue'
import BaseButton from '@/components/base/BaseButton.vue'

import {useUploadAttachmentsMutation, useDeleteAttachmentMutation} from '@/client/queries/attachments'
import {useConfigStore} from '@/stores/config'
import {uploadSizeError} from '@/helpers/uploadSize'
import {canPreviewAudio, canPreviewImage, previewKind, type PreviewKind} from '@/helpers/attachmentPreview'
import {getDisplayName} from '@/helpers/user'
import type {TaskAttachment as IAttachment} from '@/client/generated'
import type {Task as ITask} from '@/client/generated'

import {formatDateLong} from '@/helpers/time/formatDate'
import {downloadAttachment, fetchAttachmentUrl, generateAttachmentUrl, releaseAttachmentUrl} from '@/helpers/attachments'
import {downloadBlob} from '@/helpers/downloadBlob'
import {getHumanSize} from '@/helpers/getHumanSize'
import {useCopyToClipboard} from '@/composables/useCopyToClipboard'
import {error, success} from '@/message'
import {isRequestContextAbort} from '@/client/requestContext'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'
import {useI18n} from 'vue-i18n'
import FilePreview from '@/components/tasks/partials/FilePreview.vue'
import ImageLightbox from '@/components/misc/ImageLightbox.vue'
import AudioPreview from '@/components/tasks/partials/AudioPreview.vue'

type AudioPreviewInstance = InstanceType<typeof AudioPreview>

const props = withDefaults(defineProps<{
	task: ITask,
	editEnabled?: boolean,
}>(), {
	editEnabled: true,
})

const EDITOR_SELECTOR = '.tiptap, .tiptap__editor, [contenteditable]'

function eventTargetsEditor(event: Event | null | undefined): boolean {
	if (!event) {
		return false
	}

	const target = event.target
	if (target instanceof HTMLElement && target.closest(EDITOR_SELECTOR)) {
		return true
	}

	if (typeof event.composedPath === 'function') {
		return event.composedPath().some(element =>
			element instanceof HTMLElement && element.matches(EDITOR_SELECTOR),
		)
	}

	return false
}

const updateTask = useUpdateTaskMutation()
const {t} = useI18n({useScope: 'global'})

// Comment uploads are real task-owned files for authorization/storage, but
// appear as links within their originating comment rather than twice in the
// task's general attachment panel. Older files default to task context.
const attachments = computed(() => (props.task.attachments ?? [])
	.filter(a => (a as IAttachment & {source?: string}).source !== 'comment'))
const configStore = useConfigStore()
const uploadMutation = useUploadAttachmentsMutation()
const deleteMutation = useDeleteAttachmentMutation()
const uploadBatch = ref<{
	completed: number,
	total: number,
	currentPercent?: number,
} | null>(null)
// the fetch client reports no byte progress, so the first file can only be shown as indeterminate
const uploadProgress = computed(() => uploadBatch.value === null ? undefined
	: ((uploadBatch.value.completed + (uploadBatch.value.currentPercent ?? 0) / 100) / uploadBatch.value.total) * 100)
const loading = computed(() => uploadBatch.value !== null
	|| uploadMutation.isPending.value
	|| deleteMutation.isPending.value
	|| updateTask.isPending.value)

const isDraggingFiles = ref(false)
const isDragOverEditor = ref(false)

function resetDragState() {
	isDraggingFiles.value = false
	isDragOverEditor.value = false
}

/**
 * Check if a drag event contains actual files (not text being dragged).
 * This prevents the file upload overlay from appearing when dragging text
 * from within the editor to outside it.
 */
function eventContainsFiles(event: Event | null | undefined): boolean {
	if (!event || !(event instanceof DragEvent)) {
		return false
	}
	return event.dataTransfer?.types.includes('Files') ?? false
}

const {isOverDropZone} = useDropZone(document, {
	onEnter(files, event) {
		if (!props.editEnabled) {
			return
		}

		// Only show dropzone if actual files are being dragged, not text
		if (!eventContainsFiles(event)) {
			return
		}

		isDraggingFiles.value = true
		isDragOverEditor.value = eventTargetsEditor(event)
	},
	onOver(files, event) {
		if (!props.editEnabled) {
			return
		}

		isDragOverEditor.value = eventTargetsEditor(event)
	},
	onLeave(files, event) {
		if (!props.editEnabled) {
			return
		}

		if (!isOverDropZone.value) {
			resetDragState()
			return
		}

		isDragOverEditor.value = eventTargetsEditor(event)
	},
	onDrop(files, event) {
		if (!props.editEnabled) {
			return
		}

		const dropOverEditor = eventTargetsEditor(event)
		resetDragState()

		// Ignore drops over editor - let TipTap handle them
		if (dropOverEditor || !files || files.length === 0) {
			return
		}

		uploadFilesToTask(files)
	},
})

const showDropzone = computed(() =>
	props.editEnabled && isDraggingFiles.value && !isDragOverEditor.value,
)

// A <dialog> opened with showModal() (e.g. the Kanban task detail) renders in
// the browser's top layer, so the full-screen dropzone overlay teleported to
// <body> would paint behind it regardless of z-index. Teleport it into the
// topmost open dialog instead, mirroring Notification.vue.
const dropzoneTeleportTarget = ref<string | HTMLElement>('body')
let dialogObserver: MutationObserver | null = null

function syncDropzoneTeleportTarget() {
	const dialogs = document.querySelectorAll<HTMLDialogElement>('dialog.modal-dialog[open]')
	dropzoneTeleportTarget.value = dialogs.item(dialogs.length - 1) ?? 'body'
}

onMounted(() => {
	syncDropzoneTeleportTarget()
	dialogObserver = new MutationObserver(syncDropzoneTeleportTarget)
	dialogObserver.observe(document.body, {
		attributes: true,
		attributeFilter: ['open'],
		childList: true,
		subtree: true,
	})
})

onBeforeUnmount(() => {
	dialogObserver?.disconnect()
	dialogObserver = null
})

watch(() => props.editEnabled, enabled => {
	if (!enabled) {
		resetDragState()
	}
})

function attachmentMetaTooltip(attachment: IAttachment): string {
	const createdBy = t('task.attachment.createdBy', [
		formatDateLong(attachment.created),
		getDisplayName(attachment.created_by ?? {}),
	])

	return attachment.file?.mime
		? `${attachment.file?.mime} · ${createdBy}`
		: createdBy
}

const filesRef = ref<HTMLInputElement | null>(null)

function openFilePicker() {
	if (loading.value) {
		return
	}

	filesRef.value?.click()
}

async function uploadNewAttachment() {
	const input = filesRef.value
	const files = input?.files

	if (files && files.length > 0) {
		await uploadFilesToTask(files)
	}

	// without this, re-picking the same file fires no change event
	if (input) input.value = ''
}

async function uploadFilesToTask(files: File[] | FileList) {
	if (loading.value) {
		return
	}

	const taskId = props.task.id!
	const batch = Array.from(files)
	uploadBatch.value = {
		completed: 0,
		total: batch.length,
	}
	for (const [index, file] of batch.entries()) {
		const sizeMessage = uploadSizeError(file, configStore.max_file_size)
		if (sizeMessage) {
			error(new Error(sizeMessage))
			uploadBatch.value = {completed: index + 1, total: batch.length}
			continue
		}
		try {
			const result = await uploadMutation.mutateAsync({taskId, files: [file], onProgress: (loaded, total) => {
				if (uploadBatch.value !== null && total > 0) uploadBatch.value = {...uploadBatch.value, currentPercent: Math.min(100, loaded / total * 100)}
			}})
			for (const err of result.errors ?? []) error(err)
		} catch {
			// mutation already toasts; keep uploading the remaining files
		}
		uploadBatch.value = {
			completed: index + 1,
			total: batch.length,
		}
	}
	// render the finished bar before it disappears
	await nextTick()
	uploadBatch.value = null
}

const attachmentToDelete = ref<IAttachment | null>(null)

function setAttachmentToDelete(attachment: IAttachment | null) {
	attachmentToDelete.value = attachment
}

async function deleteAttachment() {
	if (attachmentToDelete.value === null) {
		return
	}

	const target = attachmentToDelete.value
	try {
		await deleteMutation.mutateAsync({taskId: props.task.id!, id: target.id!})
	} catch {
		return
	}
	if (attachmentToDelete.value === target) setAttachmentToDelete(null)
}

interface Preview {
	kind: PreviewKind
	blobUrl: string
	name: string
}

const preview = ref<Preview | null>(null)
const previewLoading = ref(false)
const previewFailed = ref(false)
let previewRequestToken = 0

function replacePreview(next: Preview | null) {
	releaseAttachmentUrl(preview.value?.blobUrl)
	preview.value = next
}

function closePreview() {
	// an in-flight blob must not re-open the dismissed modal
	previewRequestToken++
	replacePreview(null)
	previewLoading.value = false
	previewFailed.value = false
}

// a detached <video> can still fire error after its blob url was revoked
function onVideoError(e: Event) {
	if ((e.target as HTMLVideoElement).src !== preview.value?.blobUrl) {
		return
	}
	previewFailed.value = true
}

function downloadVideoPreview() {
	const current = preview.value
	if (current === null) {
		return
	}

	// downloadBlob revokes the url itself, so hand over ownership before closing
	preview.value = null
	downloadBlob(current.blobUrl, current.name)
	closePreview()
}

onBeforeUnmount(closePreview)

const audioPlayers = new Map<IAttachment['id'], AudioPreviewInstance>()

function setAudioPlayerRef(attachment: IAttachment, el: Element | ComponentPublicInstance | null) {
	if (el === null) {
		audioPlayers.delete(attachment.id!)
		return
	}

	audioPlayers.set(attachment.id, el as AudioPreviewInstance)
}

async function download(attachment: IAttachment) {
	try {
		await downloadAttachment(attachment)
	} catch (e) {
		if (!isRequestContextAbort(e)) {
			error(e)
		}
	}
}

async function viewOrDownload(attachment: IAttachment) {
	if (canPreviewAudio(attachment) && audioPlayers.has(attachment.id!)) {
		await audioPlayers.get(attachment.id!)!.play()
		return
	}

	const kind = previewKind(attachment)
	if (kind === null) {
		await download(attachment)
		return
	}

	closePreview()

	previewRequestToken++
	const requestToken = previewRequestToken

	// only video is big enough that the full-buffer wait reads as a dead click
	previewLoading.value = kind === 'video'

	try {
		const blobUrl = await fetchAttachmentUrl({id: attachment.id!, task_id: attachment.task_id!})
		// stale response: release without assigning, the img may still be decoding the current url
		if (requestToken !== previewRequestToken) {
			releaseAttachmentUrl(blobUrl)
			return
		}
		previewLoading.value = false
		replacePreview({kind, blobUrl, name: attachment.file?.name ?? ''})
	} catch (e) {
		if (requestToken !== previewRequestToken) {
			return
		}
		previewLoading.value = false
		if (!isRequestContextAbort(e)) {
			error(e)
		}
	}
}

const copy = useCopyToClipboard()

function copyUrl(attachment: IAttachment) {
	copy(generateAttachmentUrl(props.task.id!, attachment.id!))
}

async function setCoverImage(attachment: IAttachment | null) {
	await updateTask.mutateAsync({...props.task, id: props.task.id!, cover_image_attachment_id: attachment?.id ?? 0})
	success({message: t('task.attachment.successfullyChangedCoverImage')})
}

defineExpose({openFilePicker})
</script>

<style lang="scss" scoped>
.attachment-upload-progress {display:flex; align-items:center; gap:.6rem; margin-block:.4rem .8rem; font-size:.8rem;}
.attachment-upload-progress > span {white-space:nowrap;}
.attachment-upload-progress :deep(progress) {inline-size:100%;}

.attachments {
	input[type="file"] {
		display: none;
	}

	@media screen and (max-width: $tablet) {
		.button {
			inline-size: 100%;
		}
	}
}

.files {
	margin-block-end: 1rem;
}

.attachment {
	@media screen and (max-width: $tablet) {
		grid-template-columns: 2rem minmax(0, 1fr);
		gap: .5rem;
	}
	display: grid;
	grid-template-columns: 2.25rem minmax(0, 1fr);
	align-items: center;
	inline-size: 100%;
	
	gap: .55rem;
	padding: .35rem .55rem;
	margin-block: .25rem;
	transition: background-color $transition, border-color $transition;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 7px;

	&:hover {
		background-color: var(--grey-200);
	}
}

.attachment-open {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	inline-size: 100%;
	min-inline-size: 0;
	padding: 0;
	border: 0;
	background: transparent;
	color: inherit;
	font: inherit;
	text-align: start;
	cursor: pointer;
}

.filename {
	display: flex;
	align-items: center;
	font-weight: 500;
	font-size: .82rem;
	min-block-size: 1.1rem;
	color: var(--text);
	text-align: start;
	word-break: break-all;
	min-inline-size: 0;
}

.attachment-info-meta,
.attachment-actions {
	color: var(--grey-500);
	font-size: .79rem;
}

.attachment-actions {
	display: flex;
	margin-block-start: .1rem;
	margin-block-end: 0;
	gap: .3rem;
}

.dropzone {
	position: fixed;
	background: hsla(var(--grey-100-hsl), 0.8);
	inset-block-start: 0;
	inset-inline-start: 0;
	inset-block-end: 0;
	inset-inline-end: 0;
	z-index: 4001; // above app chrome when teleported to body (no modal open)
	text-align: center;

	&.hidden {
		display: none;
	}
}

.drop-hint {
	position: absolute;
	inset-block-end: 0;
	inset-inline-start: 0;
	inset-inline-end: 0;

	.icon {
		inline-size: 100%;
		font-size: 5rem;
		block-size: auto;
		text-shadow: var(--shadow-md);
		animation: bounce 2s infinite;

		@media (prefers-reduced-motion: reduce) {
			animation: none;
		}
	}

	.hint {
		margin: .5rem auto 2rem;
		border-radius: $radius;
		box-shadow: var(--shadow-md);
		background: var(--primary);
		padding: 1rem;
		color: $white; // Should always be white because of the background, regardless of the theme
		inline-size: 100%;
		max-inline-size: 300px;
	}
}

.attachment-info-column {
	display: flex;
	flex-flow: column wrap;
	align-self: start;
	min-inline-size: 0;
}

.attachment-info-meta {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: .35rem .55rem;
	margin-block: .15rem 0;
	font-size: .73rem;
	color: var(--text-light);

	> span {
		padding: 0 .25rem;
	}

	:deep(.avatar-wrapper) {
		margin-inline-end: 0;
	}
}

.attachment-uploader {font-weight: 500; color: var(--text-strong);}
.attachment-created {font-variant-numeric: tabular-nums;}
.attachment-size {
	padding: .15rem .45rem !important;
	background: var(--grey-50);
	border: 1px solid var(--card-border-color);
	border-radius: 6px;
}
.attachment-delete-action {
	border-radius: 6px;
	&:hover, &:focus-visible {
		background: color-mix(in srgb, var(--danger) 12%, var(--white));
		color: var(--danger);
	}
}

.attachment-info-meta-button {
	color: var(--link);
	padding: 0 .25rem;
}

@keyframes bounce {
	0%,
	20%,
	53%,
	80%,
	100% {
		animation-timing-function: cubic-bezier(0.215, 0.61, 0.355, 1);
		transform: translate3d(0, 0, 0);
	}

	40%,
	43% {
		animation-timing-function: cubic-bezier(0.755, 0.05, 0.855, 0.06);
		transform: translate3d(0, -30px, 0);
	}

	70% {
		animation-timing-function: cubic-bezier(0.755, 0.05, 0.855, 0.06);
		transform: translate3d(0, -15px, 0);
	}

	90% {
		transform: translate3d(0, -4px, 0);
	}
}

.preview-column {
	max-inline-size: 2.25rem;
	block-size: 2.25rem;
}

// Redundant mouse-only click target; the real control is button.attachment-open.
.preview-open {
	display: block;
	inline-size: 100%;
	block-size: 100%;
	padding: 0;
	border: 0;
	background: transparent;
	cursor: pointer;
}

.attachment-preview {
	block-size: 100%;
}

.pdf-preview-iframe {
	inline-size: 100%;
	max-inline-size: calc(100% - 4rem);
	block-size: calc(100vh - 40px);
	border: none;
	margin: 0 auto;
	display: block;
}

.video-preview {
	inline-size: auto;
	max-inline-size: calc(100% - 4rem);
	max-block-size: calc(100vh - 40px);
	margin: 0 auto;
	display: block;
}

// unlike the video and iframe branches, the error state has no opaque media of its own to sit on
.video-preview-error {
	max-inline-size: 25rem;
	margin: 0 auto;
	padding: 2rem;
	border-radius: $radius;
	background: var(--white);
	color: var(--text);
	text-align: center;

	p {
		margin-block-end: 1rem;
	}
}

.is-task-cover {
	background: var(--primary);
	color: var(--white);
	margin-inline-start: .25rem;
	padding: .25rem .35rem;
	border-radius: 4px;
	font-size: .75rem;
}

/* GitHub-like attachment rows with compact metadata and inline actions. */
.files {
	border: 1px solid var(--card-border-color);
	border-radius: 8px;
	overflow: hidden;
	margin-block-end: .75rem;
}
.files .attachment {
	margin: 0;
	border: 0;
	border-radius: 0;
	padding: .4rem .65rem;
	min-block-size: 3rem;
}
.files .attachment + .attachment {
	border-block-start: 1px solid var(--card-border-color);
}
.files .attachment:hover {
	background: var(--grey-50);
}
.attachment-info-column {
	overflow: hidden;
}
.attachment-info-meta {
	gap: .15rem .45rem;
	margin: .1rem 0 0;
}
.attachment-actions {
	gap: .2rem;
}
.attachment-info-meta-button {
	border-radius: 5px;
	min-inline-size: 1.5rem;
	min-block-size: 1.5rem;
}
.attachment-info-meta-button:hover,
.attachment-info-meta-button:focus-visible {
	background: var(--grey-100);
}

/* File list: filename and metadata on the left, stable controls on the right. */
.files .attachment {
	grid-template-columns: 2rem minmax(0, 1fr) auto;
	gap: .6rem;
	padding: .4rem .65rem;
	min-block-size: 3rem;
}
.preview-column {inline-size: 2rem; block-size: 2rem; max-inline-size: 2rem; min-inline-size: 0; overflow: hidden;}
.attachment-preview {display: block; inline-size: 100%; block-size: 100%; max-inline-size: 100%; max-block-size: 100%; overflow: hidden; object-fit: contain;}
.attachment-info-column {align-self: center; min-inline-size: 0;}
.filename {display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; word-break: normal; font-size: .84rem; line-height: 1.25;}
.attachment-info-meta {font-size: .7rem; line-height: 1.3; gap: .1rem .35rem;}
.attachment-row-actions {display:flex; align-items:center; justify-content:flex-end; flex-shrink:0;}
.attachment-row-actions .attachment-actions {display:flex; align-items:center; flex-wrap:nowrap; gap:.1rem; margin:0;}
.attachment-info-meta-button {display:inline-flex; align-items:center; justify-content:center; min-inline-size:1.8rem; min-block-size:1.8rem; padding:.2rem; color:var(--text-light);}
.attachment-delete-action {color:var(--danger);}
@media screen and (max-width: $tablet) {
	.files .attachment {grid-template-columns: 1.8rem minmax(0, 1fr);}
	.attachment-row-actions {grid-column: 2; justify-content:flex-start;}
}
</style>
