<template>
	<div
		v-if="enabled"
		ref="commentsRef"
		class="content details comments-container"
	>
		<h2
			v-if="canWrite || comments.length > 0"
			class="comments-heading task-section-title"
			:class="{'d-print-none': comments.length === 0}"
		>
			<span>
				<span class="icon is-grey">
					<Icon :icon="['far', 'comments']" />
				</span>
				{{ $t('task.comment.title') }}
			</span>
			<BaseButton
				v-if="comments.length > 0"
				class="comment-sort-button"
				@click="toggleSortOrder"
			>
				<Icon :icon="commentSortOrder === 'asc' ? 'arrow-down-short-wide' : 'arrow-up-short-wide'" />
				{{ commentSortOrder === 'asc' ? $t('task.comment.sortOldestFirst') : $t('task.comment.sortNewestFirst') }}
			</BaseButton>
		</h2>
		<div class="comments">
			<span
				v-if="listPending"
				class="is-flex is-align-items-center mbs-4 mbe-4 mis-2"
			>
				<span class="loader is-inline-block mie-2" />
				{{ $t('task.comment.loading') }}
			</span>
			<div
				v-for="c in comments"
				:id="`comment-${c.id}`"
				:key="c.id"
				class="media comment posted-comment"
			>
				<figure class="media-left is-hidden-mobile">
					<UserAvatar
						:user="c.author"
						:size="48"
						class="image is-avatar"
					/>
					<figcaption class="is-sr-only">
						{{ $t('misc.avatarOfUser', {user: getDisplayName(c.author)}) }}
					</figcaption>
				</figure>
				<div class="media-content">
					<div class="comment-info">
						<UserAvatar
							:user="c.author"
							:size="20"
							class="image is-avatar d-print-none"
						/>
						<strong>{{ getDisplayName(c.author) }}</strong>
						<span
							v-tooltip="formatDateLong(c.created)"
							class="has-text-grey"
						>
							{{ formatDisplayDate(c.created) }}
						</span>
						<span
							v-if="c.created && c.updated && +new Date(c.created) !== +new Date(c.updated)"
							v-tooltip="formatDateLong(c.updated)"
						>
							· {{ $t('task.comment.edited', {date: formatDisplayDate(c.updated)}) }}
						</span>
						<a
							v-tooltip="$t('task.comment.permalink')"
							:href="`#comment-${c.id}`"
							class="comment-permalink"
							:title="$t('task.comment.permalink')"
							@click.prevent.stop="copy(getCommentUrl(`${c.id}`))"
						>
							<span class="is-sr-only">{{ $t('task.comment.permalink') }}</span>
							<Icon icon="link" />
						</a>
						<CustomTransition name="fade">
							<span
								v-if="
									showLoading &&
										saving === c.id
								"
								class="is-inline-flex"
							>
								<span class="loader is-inline-block mie-2" />
								{{ $t('misc.saving') }}
							</span>
							<span
								v-else-if="
									!showLoading &&
										saved === c.id
								"
								class="has-text-success"
							>
								{{ $t('misc.saved') }}
							</span>
						</CustomTransition>
						<ChangeHistory
							v-if="c.id > 0 && c.created && c.updated && +new Date(c.created) !== +new Date(c.updated) && !authStore.isLinkShareAuth"
							kind="comment"
							:entity-id="c.id"
							:task-id="taskId"
							compact
						/>
					</div>
					<Editor
						:model-value="commentDrafts[c.id] ?? c.comment"
						:is-edit-enabled="canWrite && c.author?.id === currentUserId"
						:upload-callback="attachmentUpload"
						:upload-enabled="true"
						:bottom-actions="actions[c.id]"
						:show-save="true"
						:show-cancel="true"
						:enable-discard-shortcut="true"
						:enable-mentions="true"
						:project-id="projectId"
						initial-mode="preview"
						@update:modelValue="value => changeComment(c, value)"
						@save="() => {
							captureEditDraft(c)
							editComment(taskId)
						}"
						@discard="discardCommentEdit(c)"
					/>
					<Reactions 
						:model-value="c.reactions"
						class="mbs-2 d-print-none"
						entity-kind="comments"
						:entity-id="c.id"
						:task-id="taskId"
						:disabled="!canWrite"
					/>
				</div>
			</div>

			<PaginationEmit
				v-if="totalPages > 1"
				:total-pages="totalPages"
				:current-page="currentPage"
				@pageChanged="changePage"
			/>

			<div
				v-if="canWrite"
				class="media comment comment-composer d-print-none"
				:class="{'new-comment-top': commentSortOrder === 'desc'}"
			>
				<figure class="media-left is-hidden-mobile">
					<UserAvatar
						:user="authStore.info"
						:size="48"
						class="image is-avatar"
					/>
					<figcaption class="is-sr-only">
						{{ $t('misc.avatarOfUser', {user: getDisplayName(authStore.info)}) }}
					</figcaption>
				</figure>
				<div class="media-content">
					<p class="composer-label"><Icon :icon="['far', 'comments']" /> 编写新评论</p>
					<div class="form">
						<CustomTransition name="fade">
							<span
								v-if="showLoading && creating"
								class="is-inline-flex"
							>
								<span class="loader is-inline-block mie-2" />
								{{ $t('task.comment.creating') }}
							</span>
						</CustomTransition>
						<div class="field">
							<Editor
								v-if="editorActive"
								:key="taskId"
								ref="newCommentEditor"
								v-model="newCommentText"
								:class="{
									'is-loading':
										showLoading &&
										!isCommentEdit,
								}"
								:upload-callback="attachmentUpload"
								:placeholder="$t('task.comment.placeholder')"
								:enable-mentions="true"
								:project-id="projectId"
								:storage-key="commentStorageKey"
								@save="addComment()"
							/>
						</div>
						<div class="field comment-composer-actions">
							<input
								ref="commentFilesInput"
								type="file"
								multiple
								class="is-hidden"
								:aria-label="$t('task.attachment.upload')"
								@change="uploadFilesFromComment"
							>
							<XButton
								type="button"
								:loading="loading && !isCommentEdit"
								:disabled="!newCommentText || creating || commentUploadPending"
								@click="addComment()"
							>
								{{ $t('task.comment.comment') }}
							</XButton>
							<XButton
								type="button"
								variant="secondary"
								icon="paperclip"
								:loading="commentUploadPending"
								:disabled="commentUploadPending || creating"
								@click="commentFilesInput?.click()"
							>
								{{ $t('task.attachment.upload') }}
							</XButton>
						</div>
						<p v-if="commentUploadError" class="help is-danger" role="alert">{{ commentUploadError }}</p>
					</div>
				</div>
			</div>
		</div>


		<Modal
			:enabled="showDeleteModal"
			@close="showDeleteModal = false"
			@submit="() => deleteComment(commentToDelete)"
		>
			<template #header>
				<span>{{ $t('task.comment.delete') }}</span>
			</template>

			<template #text>
				<p>
					{{ $t('task.comment.deleteText1') }}<br>
					<strong class="has-text-white">{{ $t('misc.cannotBeUndone') }}</strong>
				</p>
			</template>
		</Modal>
	</div>
</template>

<script setup lang="ts">
import {useUpdateFrontendSettingsMutation} from '@/client/queries/account'
import {ref, computed, nextTick, provide, watch, onBeforeUnmount} from 'vue'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import CustomTransition from '@/components/misc/CustomTransition.vue'
import Editor from '@/components/input/AsyncEditor'
import PaginationEmit from '@/components/misc/PaginationEmit.vue'
import UserAvatar from '@/components/misc/UserAvatar.vue'

import {useQuery} from '@tanstack/vue-query'
import {
	commentsQuery,
	useCreateCommentMutation,
	useUpdateCommentMutation,
	useDeleteCommentMutation,
	type CommentResponse,
} from '@/client/queries/comments'

import type {Task as ITask} from '@/client/generated'

import {generateAttachmentUrl} from '@/helpers/attachments'
import {useUploadAttachmentsMutation} from '@/client/queries/attachments'
import ChangeHistory from '@/components/history/ChangeHistory.vue'
import {getErrorText} from '@/message'
import {formatDateLong, formatDisplayDate} from '@/helpers/time/formatDate'
import {clearEditorDraft} from '@/helpers/editorDraftStorage'
import {getDisplayName} from '@/helpers/user'
import {useConfigStore} from '@/stores/config'
import {useAuthStore} from '@/stores/auth'
import Reactions from '@/components/input/Reactions.vue'
import {useCopyToClipboard} from '@/composables/useCopyToClipboard'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {commentReplyContextKey, scrollAndHighlightComment} from '@/components/tasks/partials/commentReplyContext'

const props = withDefaults(defineProps<{
	taskId: number,
	projectId: number,
	canWrite?: boolean,
}>(), {
	canWrite: true,
})

const copy = useCopyToClipboard()

const {t} = useI18n({useScope: 'global'})
const configStore = useConfigStore()
const authStore = useAuthStore()
const updateFrontendSettings = useUpdateFrontendSettingsMutation()

const localSortOrder = ref<'asc' | 'desc' | null>(null)
const commentSortOrder = computed(() => localSortOrder.value ?? authStore.settings.frontend_settings.comment_sort_order ?? 'asc')

const currentPage = ref(1)
const commentQuery = useQuery(computed(() => ({
	...commentsQuery(props.taskId, commentSortOrder.value, currentPage.value),
	enabled: configStore.task_comments_enabled && props.taskId > 0,
})))
const comments = computed(() => commentQuery.data.value?.items ?? [])
const totalPages = computed(() => commentQuery.data.value?.total_pages ?? 0)
const createMutation = useCreateCommentMutation()
const updateMutation = useUpdateCommentMutation()
const deleteMutation = useDeleteCommentMutation()
const listPending = useDelayedLoading(commentQuery.isLoading)
const loading = computed(() => createMutation.isPending.value || updateMutation.isPending.value)
const showLoading = useDelayedLoading(loading)
const commentDrafts = ref<Record<number, string>>({})

const showDeleteModal = ref(false)
const commentToDelete = ref<number | null>(null)

const isCommentEdit = ref(false)
const commentEdit = ref<CommentResponse | null>(null)

const newCommentText = ref('')

const saved = ref<ITask['id'] | null>(null)
const saving = ref<ITask['id'] | null>(null)

const currentUserId = computed(() => authStore.info?.id ?? null)
const enabled = computed(() => configStore.task_comments_enabled)
const actions = computed(() => {
	if (!props.canWrite) {
		return {}
	}
	return Object.fromEntries(comments.value.map((comment) => {
		const list: {action: () => void, title: string, danger?: boolean}[] = [{
			action: () => startReplyTo(comment),
			title: t('task.comment.reply'),
		}]
		if (comment.author?.id === currentUserId.value || authStore.info?.is_admin === true) {
			list.push({
				action: () => toggleDelete(comment.id),
				title: t('misc.delete'),
				danger: true,
			})
		}
		return [comment.id, list]
	}))
})

const commentStorageKey = computed(() => `task-comment-${props.taskId}`)

const commentsRef = ref<HTMLElement | null>(null)
const newCommentEditor = ref<{setReplyContent: (html: string) => Promise<void>} | null>(null)

provide(commentReplyContextKey, {
	findComment: (id: number) => comments.value.find(c => c.id === id),
	scrollToComment: scrollAndHighlightComment,
})

// Strip <mention-user> elements from a reply quote so reposting the parent
// body doesn't trigger fresh notifications for users mentioned in the
// original. The inner text is kept so the quote still reads correctly.
function stripMentionsForQuote(html: string): string {
	if (!html) {
		return ''
	}
	const doc = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html')
	doc.querySelectorAll('mention-user').forEach((el) => {
		const label = (el.getAttribute('data-label') ?? el.textContent ?? '').trim()
		el.replaceWith(label ? `@${label.replace(/^@+/, '')}` : '')
	})
	return doc.body.firstElementChild?.innerHTML ?? ''
}

async function startReplyTo(parent: CommentResponse) {
	const body = stripMentionsForQuote(parent.comment ?? '')
	const draft = `<blockquote data-comment-id="${parent.id}">${body}</blockquote><p></p>`
	if (!editorActive.value) {
		editorActive.value = true
	}
	// Editor mounts asynchronously through defineAsyncComponent; wait until
	// the ref is populated before pushing content in. Bail with a warning
	// rather than fall back to `newCommentText = draft` — the modelValue
	// watcher in TipTap.vue would land the editor in preview mode, leaving
	// the user unable to type without clicking the editor first.
	const editor = await waitForEditorRef()
	if (!editor) {
		console.warn('Reply editor did not mount in time; aborting reply prefill.')
		return
	}
	await editor.setReplyContent(draft)
}

async function waitForEditorRef() {
	const start = performance.now()
	while (!newCommentEditor.value && performance.now() - start < 2000) {

		await nextTick()
	}
	return newCommentEditor.value
}


// The rich-text toolbar inserts images, but non-image files belong in task
// attachments. This explicit action uploads any file and inserts a link in
// the comment draft; upload alone never publishes a comment.
const commentFilesInput = ref<HTMLInputElement | null>(null)
const commentUploadPending = ref(false)
const commentUploadError = ref('')

function escapeCommentHTML(value: string): string {
	return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

async function uploadFilesFromComment(event: Event) {
	const input = event.target as HTMLInputElement
	const files = Array.from(input.files ?? [])
	input.value = '' // re-selecting the same file must remain possible
	if (files.length === 0 || commentUploadPending.value || creating.value) return

	const taskId = props.taskId
	commentUploadError.value = ''
	commentUploadPending.value = true
	try {
		for (const file of files) {
			const result = await uploadAttachments.mutateAsync({taskId, files: [file], source: 'comment'})
			if (props.taskId !== taskId) return
			const attachment = result.success?.[0]
			if (!attachment?.id) {
				commentUploadError.value = '部分文件上传失败，请检查文件大小和附件权限。'
				continue
			}
			const url = generateAttachmentUrl(taskId, attachment.id)
			const link = '<p><a href="' + escapeCommentHTML(url) + '">' +
				escapeCommentHTML(file.name) + '</a></p>'
			// The TipTap component switches external model changes to preview mode.
			// Use its existing exposed method to preserve draft + edit focus.
			const html = newCommentText.value + link
			if (newCommentEditor.value) {
				await newCommentEditor.value.setReplyContent(html)
			} else {
				newCommentText.value = html
			}
		}
	} catch (cause) {
		if (props.taskId === taskId) commentUploadError.value = getErrorText(cause) || '评论附件上传失败，请检查权限、容量和网络。'
	} finally {
		commentUploadPending.value = false
	}
}

// the editor toasts the rejection itself, a mutation toast would duplicate it
const uploadAttachments = useUploadAttachmentsMutation(() => false)

function attachmentUpload(files: File[] | FileList): Promise<string[]> {
	commentUploadError.value = ''
	return Promise.all(Array.from(files).map(async file => {
		const result = await uploadAttachments.mutateAsync({
			taskId: props.taskId,
			files: [file],
			source: 'comment',
		}).catch((cause: unknown) => {
			commentUploadError.value = getErrorText(cause) || '评论附件上传失败'
			throw cause
		})
		const [uploaded] = result.success ?? []
		// forwarded verbatim: the editor's toast translates the error code, which a rewrapped message would lose
		if (uploaded?.id === undefined) throw result.errors?.[0] ?? new Error('Attachment upload returned no file')
		return generateAttachmentUrl(props.taskId, uploaded.id)
	}))
}

async function changePage(page: number) {
	commentsRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start', inline: 'nearest' })
	currentPage.value = page
}

async function toggleSortOrder() {
	const newOrder = commentSortOrder.value === 'asc' ? 'desc' : 'asc'
	const session = authStore.session
	if (session && !authStore.isLinkShareAuth) {
		localSortOrder.value = newOrder
		try {
			await updateFrontendSettings.mutateAsync({
				id: session.id,
				type: session.type,
				frontendSettings: {comment_sort_order: newOrder},
			})
		} catch {
			return
		} finally {
			localSortOrder.value = null
		}
	} else {
		localSortOrder.value = newOrder
	}
	currentPage.value = 1
}

watch(() => props.taskId, () => {
	currentPage.value = 1
	newCommentText.value = ''
	commentUploadError.value = ''
	commentDrafts.value = {}
	commentEdit.value = null
	commentToDelete.value = null
	showDeleteModal.value = false
})

const editorActive = ref(true)
const creating = ref(false)

async function addComment() {
	if (!newCommentText.value || creating.value) return
	const taskId = props.taskId
	const text = newCommentText.value
	creating.value = true
	try {
		await createMutation.mutateAsync({taskId, comment: text})
		if (props.taskId !== taskId) return
		currentPage.value = commentSortOrder.value === 'desc' ? 1 : Math.max(1, totalPages.value)
		if (newCommentText.value === text) {
			newCommentText.value = ''
			clearEditorDraft(commentStorageKey.value)
		}
		if (commentSortOrder.value === 'desc') commentsRef.value?.scrollIntoView({behavior: 'smooth', block: 'start'})
	} catch {
		return
	} finally {
		creating.value = false
	}
}

function captureEditDraft(comment: CommentResponse) {
	isCommentEdit.value = true
	commentEdit.value = {...comment, comment: commentDrafts.value[comment.id] ?? comment.comment}
}

function changeComment(comment: CommentResponse, text: string) {
	// Comment edits are explicit-save only. An automatic 5 s save would make
	// "取消编辑" unable to guarantee that the draft is discarded.
	commentDrafts.value[comment.id] = text
	captureEditDraft(comment)
}

function discardCommentEdit(comment: CommentResponse) {
	// TipTap has already restored its original content without emitting an
	// editor change. Remove the reactive draft so preview shows the server copy.
	delete commentDrafts.value[comment.id]
	if (commentEdit.value?.id === comment.id) commentEdit.value = null
	isCommentEdit.value = false
}

function toggleDelete(id: number) {
	commentToDelete.value = id
	showDeleteModal.value = true
}

let savedTimeout: ReturnType<typeof setTimeout> | undefined
onBeforeUnmount(() => {
	clearTimeout(savedTimeout)
})

async function editComment(taskId: number) {
	const draft = commentEdit.value
	if (!draft?.comment) {
		isCommentEdit.value = false
		return
	}
	saving.value = draft.id
	try {
		await updateMutation.mutateAsync({taskId, id: draft.id, comment: draft.comment})
		if (props.taskId !== taskId) return
		if (commentDrafts.value[draft.id] === draft.comment) delete commentDrafts.value[draft.id]
		saved.value = draft.id
		clearTimeout(savedTimeout)
		savedTimeout = setTimeout(() => { saved.value = null }, 2000)
	} catch {
		return
	} finally {
		isCommentEdit.value = false
		saving.value = null
	}
}

async function deleteComment(id: number | null) {
	if (id === null) return
	const taskId = props.taskId
	try {
		await deleteMutation.mutateAsync({taskId, id})
	} catch {
		return
	}
	if (props.taskId === taskId && commentToDelete.value === id) {
		showDeleteModal.value = false
		commentToDelete.value = null
		currentPage.value = Math.min(currentPage.value, Math.max(1, totalPages.value))
	}
}

function getCommentUrl(commentId: string) {
	const frontendUrl = configStore.frontend_url
	const baseUrl = frontendUrl.endsWith('/') ? frontendUrl.slice(0, -1) : frontendUrl
	const url = new URL(location.pathname + location.search, baseUrl)
	url.hash = `comment-${commentId}`
	return url.toString()
}
</script>

<style lang="scss" scoped>
// Each saved comment and the editor are visually independent surfaces.
// This prevents an attachment link and the composer below it from reading as
// part of the same comment.
.comments-container {
	min-inline-size: 0;
}
.comments {
	gap: .9rem;
}
.comments > .media.comment {
	margin: 0 !important;
	padding: .65rem .9rem .8rem;
	border: 1px solid var(--card-border-color);
	border-radius: 10px;
	background: var(--white);
	box-shadow: 0 1px 3px rgba(15,23,42,.04);
	transition: border-color .15s ease, box-shadow .15s ease;
}
.comments > .posted-comment {
	border-color: var(--grey-300);
}
.comments > .posted-comment .comment-info {
	background: var(--grey-50);
	border: 1px solid var(--card-border-color);
	border-radius: 7px;
	padding: .62rem .8rem;
	margin-bottom: .85rem;
	min-block-size: 2.8rem;
}
.comments > .posted-comment .media-content {
	min-inline-size: 0;
}
.comments > .posted-comment :deep(.tiptap__editor) {
	padding-inline: .55rem;
	line-height: 1.65;
}
.comments > .posted-comment :deep(.tiptap__editor-actions) {
	margin-block-start: .55rem;
	padding-block: .6rem .1rem;
	border-top: 1px solid var(--card-border-color);
}
.comments > .posted-comment :deep(.tiptap__editor-actions .danger-action) {
	color: var(--danger);
}
.comments > .posted-comment :deep(.tiptap__editor-actions .danger-action:hover) {
	background: var(--danger);
	color: var(--white);
}
.comments > .posted-comment:hover {
	border-color: color-mix(in srgb, var(--primary) 22%, var(--card-border-color));
	box-shadow: 0 4px 12px rgba(15, 23, 42, .055);
}
.comments > .comment-composer {
	margin-block-start: 1.15rem !important;
	border: 1px solid color-mix(in srgb, var(--primary) 35%, var(--card-border-color));
	background: var(--grey-50);
	box-shadow: 0 3px 14px rgba(15, 23, 42, .05);
}
.comment-composer:focus-within {
	border-color: var(--primary);
	box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 14%, transparent);
}
.composer-label {
	display: flex;
	align-items: center;
	gap: .4rem;
	margin: 0 0 .8rem;
	color: var(--text-strong);
	font-size: .86rem;
	font-weight: 550;
}
.comment-info {
	padding-block-end: .6rem;
	margin-block-end: .7rem;
	border-block-end: 1px solid var(--card-border-color);
}
.comment-info strong {color: var(--text-strong); font-weight: 600;}
.comment-composer :deep(.tiptap),
.comment-composer :deep(.tiptap__editor) {
	border-radius: 8px;
}
@media screen and (max-width: $tablet) {
	.comments > .media.comment { padding: .85rem; }
}

.comment-composer-actions {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: .65rem;
	margin-top: .85rem;
	margin-bottom: 0;
}
.comment-composer :deep(.tiptap__wrapper) {
	border: 1px solid var(--grey-300);
	border-radius: 10px;
	background: var(--white);
	transition: border-color .15s ease, box-shadow .15s ease;
	&:hover { border-color: var(--grey-500); }
	&:focus-within {
		border-color: var(--primary);
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--primary) 12%, transparent);
	}
}
.comment-info > .comment-history-compact { margin-inline-start: auto; }


.media {
	align-items: flex-start;
	display: flex;
	text-align: inherit;
	padding-block-start: .5rem;

	& + .media {
		margin-block-start: .5rem;
	}
}

.media-left {
	flex-basis: auto;
	flex-grow: 0;
	flex-shrink: 0;
	margin: 0 .5rem !important;
}

.comment-info {
	display: flex;
	align-items: center;
	gap: .5rem;

	img {
		@media screen and (max-width: $tablet) {
			display: block;
			inline-size: 20px;
			block-size: 20px;
			padding-inline-end: 0;
			margin-inline-end: .5rem;
		}

		@media screen and (min-width: $tablet) {
			display: none;
		}
	}


	span,
	.comment-permalink {
		font-size: .75rem;
		line-height: 1;
	}

	.comment-permalink {
		font-size: 1rem;
		border: 1px solid transparent;
		padding: 0.25rem;
		border-radius: 1rem;
		color: var(--grey, hsl(0, 0%, 48%));
	}
	.comment-permalink:hover {
		color: var(--grey-dark, hsl(0, 0%, 29%));
		border-color: var(--grey-dark, hsl(0, 0%, 29%));
	}
}

.image.is-avatar {
	border-radius: 100%;
}

.media-content {
	flex-basis: auto;
	flex-grow: 1;
	flex-shrink: 1;
	text-align: inherit;
	inline-size: calc(100% - 48px - 2rem);
}

.comments-heading {
	display: flex;
	align-items: center;
	justify-content: space-between;
}

.comment-sort-button {
	font-size: .75rem;
	font-weight: normal;
	color: var(--grey-500);
	display: inline-flex;
	align-items: center;
	gap: .25rem;

	&:hover {
		color: var(--grey-700);
	}
}

.comments {
	display: flex;
	flex-direction: column;
}

.new-comment-top {
	order: -1;
}

.comments-container {
	scroll-margin-block-start: 4rem;
}

.media.comment {
	scroll-margin-block-start: 4rem;
	transition: background-color .3s ease-out;
	border-radius: $radius;
}

.media.comment.comment-highlight {
	background-color: hsla(var(--primary-hsl), 0.18);
	transition: background-color .15s ease-in;
}

/* GitHub-inspired discussion timeline: avatars outside bordered conversation cards. */
.comments-container {
	--discussion-border: var(--card-border-color);
	--discussion-surface: var(--white);
}
.comments {
	position: relative;
	gap: 1rem;
}
.comments::before {
	content: '';
	position: absolute;
	z-index: 0;
	inset-block: 1rem 2rem;
	inset-inline-start: 24px;
	inline-size: 2px;
	background: var(--discussion-border);
	pointer-events: none;
}
.comments > .media.comment {
	position: relative;
	z-index: 1;
	padding: 0 !important;
	border: 0 !important;
	background: transparent !important;
	box-shadow: none !important;
	overflow: visible;
	gap: .65rem;
}
.comments > .media.comment > .media-left {
	margin: 0 !important;
	flex: 0 0 48px;
	background: var(--white);
	border-radius: 50%;
}
.comments > .media.comment > .media-content {
	overflow: visible;
	min-inline-size: 0;
	inline-size: auto;
	background: var(--discussion-surface);
	border: 1px solid var(--discussion-border);
	border-radius: 8px;
	box-shadow: 0 1px 2px rgba(15, 23, 42, .04);
}
.comments > .posted-comment .comment-info {
	position: relative;
	margin: 0;
	padding: .65rem .85rem;
	border: 0;
	border-bottom: 1px solid var(--discussion-border);
	border-radius: 7px 7px 0 0;
	background: var(--grey-50);
	min-block-size: 2.5rem;
	flex-wrap: wrap;
}
.comments > .posted-comment .comment-info::before {
	content: '';
	position: absolute;
	inset-inline-start: -7px;
	inset-block-start: 16px;
	inline-size: 12px;
	block-size: 12px;
	transform: rotate(45deg);
	background: var(--grey-50);
	border-inline-start: 1px solid var(--discussion-border);
	border-block-end: 1px solid var(--discussion-border);
}
.comments > .posted-comment .media-content > :not(.comment-info) {
	margin-inline: .85rem;
}
.comments > .posted-comment .media-content > :last-child {
	margin-block-end: .75rem;
}
.comments > .posted-comment :deep(.tiptap__editor) {
	padding: .85rem .1rem;
	line-height: 1.65;
}
.comments > .comment-composer {
	margin-block-start: .75rem !important;
}
.comments > .comment-composer .media-content {
	padding: .85rem;
}
.comment-composer :deep(.tiptap__wrapper) {
	border-radius: 6px;
}
.comment-composer-actions {
	padding-block-start: .65rem;
	border-top: 1px solid var(--discussion-border);
}
@media screen and (max-width: $tablet) {
	.comments::before { display: none; }
	.comments > .media.comment > .media-left { display: none; }
	.comments > .posted-comment .comment-info::before { display: none; }
	.comments > .posted-comment .media-content > :not(.comment-info) { margin-inline: .6rem; }
}
</style>
