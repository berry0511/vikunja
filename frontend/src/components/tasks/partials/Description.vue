<template>
	<div
		:class="{'d-print-none': isEmpty}"
	>
		<h2 class="task-section-title">
			<span class="icon is-grey">
				<Icon icon="align-left" />
			</span>
			{{ $t('task.attributes.description') }}
			<CustomTransition name="fade">
				<span
					v-if="saveState === 'saving'"
					class="is-small is-inline-flex"
					aria-hidden="true"
				>
					<span class="loader is-inline-block mie-2" />
					{{ $t('misc.saving') }}
				</span>
				<span
					v-else-if="saveState === 'saved'"
					class="is-small has-text-success"
					aria-hidden="true"
				>
					<Icon icon="check" />
					{{ $t('misc.saved') }}
				</span>
			</CustomTransition>
		</h2>
		<!-- Outside the h2 so the heading keeps a stable accessible name -->
		<span
			class="is-sr-only"
			role="status"
			aria-live="polite"
		>{{ saveStateAnnouncement }}</span>
		<Editor
			v-model="description"
			class="tiptap__task-description"
			:is-edit-enabled="canWrite"
			:upload-callback="uploadCallback"
			:placeholder="$t('task.description.placeholder')"
			:show-save="true"
			:show-cancel="true"
			edit-shortcut="KeyE"
			:enable-discard-shortcut="true"
			:enable-mentions="true"
			:project-id="modelValue.project_id"
			:storage-key="descriptionStorageKey"
			@update:modelValue="markChanged"
			@save="save"
			@discard="discardDescription"
		/>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, watch, onBeforeUnmount} from 'vue'
import {onBeforeRouteLeave} from 'vue-router'
import {useI18n} from 'vue-i18n'

import CustomTransition from '@/components/misc/CustomTransition.vue'
import Editor from '@/components/input/AsyncEditor'

import { clearEditorDraft } from '@/helpers/editorDraftStorage'
import { isEditorContentEmpty } from '@/helpers/editorContentEmpty'
import {generateAttachmentUrl} from '@/helpers/attachments'
import type {Task as ITask} from '@/client/generated'
import {useUploadAttachmentsMutation} from '@/client/queries/attachments'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'

const props = defineProps<{
	modelValue: ITask,
	canWrite: boolean,
}>()

const emit = defineEmits<{
	'update:modelValue': [value: ITask]
}>()

const description = ref<string>('')
const hasChanges = ref(false)
watch(
	() => [props.modelValue.id, props.modelValue.description] as const,
	([id, value], previous) => {
		if (id === previous?.[0] && hasChanges.value) return
		description.value = value ?? ''
		hasChanges.value = false
	},
	{immediate: true},
)

const saved = ref(false)
const saving = ref(false)

const updateTask = useUpdateTaskMutation()

const {t} = useI18n({useScope: 'global'})

const savedTimeout = ref<ReturnType<typeof setTimeout> | null>(null)
const dwellTimeout = ref<ReturnType<typeof setTimeout> | null>(null)

// Saves resolve faster than "Saving…" can be read, and aria-live coalesces it away, so hold it a floor.
const MIN_SAVING_DWELL = 500
const dwelling = ref(false)

watch(saving, isSaving => {
	if (!isSaving) {
		return
	}

	dwelling.value = true
	if (dwellTimeout.value !== null) {
		clearTimeout(dwellTimeout.value)
	}
	dwellTimeout.value = setTimeout(() => {
		dwelling.value = false
	}, MIN_SAVING_DWELL)
})

const saveState = computed(() => {
	if (saving.value || dwelling.value) {
		return 'saving'
	}

	if (saved.value) {
		return 'saved'
	}

	return ''
})

// Runs from when "Saved!" reaches the screen, not from the response the floor may have delayed.
watch(saveState, state => {
	if (savedTimeout.value !== null) {
		clearTimeout(savedTimeout.value)
	}

	if (state !== 'saved') {
		return
	}

	savedTimeout.value = setTimeout(() => {
		saved.value = false
	}, 2000)
})

const saveStateAnnouncement = computed(() => {
	if (saveState.value === 'saving') {
		return t('misc.saving')
	}

	if (saveState.value === 'saved') {
		return t('misc.saved')
	}

	return ''
})

const descriptionStorageKey = computed(() => `task-description-${props.modelValue.id}`)

const isEmpty = computed(() => isEditorContentEmpty(description.value))

// An explicit save/cancel pair must never race a scheduled auto-save.
function markChanged() {
	hasChanges.value = description.value !== (props.modelValue.description ?? '')
}

function discardDescription() {
	description.value = props.modelValue.description ?? ''
	hasChanges.value = false
	clearEditorDraft(descriptionStorageKey.value)
	saved.value = false
}

onBeforeUnmount(() => {
	if (savedTimeout.value !== null) clearTimeout(savedTimeout.value)
	if (dwellTimeout.value !== null) clearTimeout(dwellTimeout.value)
})

// Do not silently save while leaving the page. Preserve the user's choice.
onBeforeRouteLeave(() => {
	if (!hasChanges.value) return true
	if (!window.confirm('任务描述有尚未保存的修改，确定放弃修改并离开吗？')) return false
	discardDescription()
	return true
})

async function save() {
	if (!hasChanges.value) {
		return
	}

	hasChanges.value = false
	saved.value = false
	saving.value = true

	try {
		const updated = await updateTask.mutateAsync({
			...props.modelValue,
			id: props.modelValue.id!,
			description: description.value,
		})
		emit('update:modelValue', updated)

		// Clear draft from localStorage when saved successfully
		clearEditorDraft(descriptionStorageKey.value)

		saved.value = true
	} catch (error) {
		// If the task was deleted (404), silently skip saving
		if ((error as {status?: number} | null)?.status === 404) {
			return
		}
		hasChanges.value = true
		// Re-throw other errors
		throw error
	} finally {
		saving.value = false
	}
}

// the editor toasts the rejection itself, a mutation toast would duplicate it
const uploadAttachments = useUploadAttachmentsMutation(() => false)

function uploadCallback(files: File[] | FileList): Promise<string[]> {
	const taskId = props.modelValue.id!
	return Promise.all(Array.from(files).map(async file => {
		const result = await uploadAttachments.mutateAsync({
			taskId,
			files: [file],
		})
		const [uploaded] = result.success ?? []
		// forwarded verbatim: the editor's toast translates the error code, which a rewrapped message would lose
		if (uploaded?.id === undefined) throw result.errors?.[0] ?? new Error('Attachment upload returned no file')
		return generateAttachmentUrl(taskId, uploaded.id)
	}))
}
</script>

<style lang="scss" scoped>

/* One consistent issue body for both tasks and Bugs; no nested oversized card. */
.tiptap__task-description {
	margin-inline-start: 0;
	:deep(.tiptap__wrapper) {
		border: 0;
		border-radius: 0;
		box-shadow: none;
		background: transparent;
		&:hover {border-color: transparent;}
		&:focus-within {border-color: transparent; box-shadow: none;}
	}
	:deep(.tiptap__editor) {
		min-block-size: 0;
		padding-block: .45rem;
		line-height: 1.6;
	}
	:deep(.tiptap__editor-actions) {
		border-top: 1px solid var(--card-border-color);
		margin-block-start: .65rem;
		padding-block-start: .65rem;
	}
}

.tiptap__task-description {
	// Always show the editor boundary even before the user focuses it.
	:deep(.tiptap__wrapper) {
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
	// The exact amount of pixels we need to make the description icon align with the buttons and the form inside the editor.
	// The icon is not exactly the same length on all sides so we need to hack our way around it.
	margin-inline-start: 4px;
}
</style>
