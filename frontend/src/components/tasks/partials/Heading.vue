<template>
	<div class="heading">
		<h1
			class="title input"
			:class="{'disabled': !canWrite}"
			:contenteditable="canWrite ? true : undefined"
			:tabindex="canWrite ? 0 : undefined"
			:aria-label="canWrite ? $t('task.attributes.title') : undefined"
			:spellcheck="false"
			@input="handleTitleInput"
			@blur="save($event.target as HTMLElement)"
			@keydown.enter.prevent.stop="!$event.isComposing && ($event.target as HTMLInputElement).blur()"
			@keydown.esc.prevent.stop="!$event.isComposing && cancel($event.target as HTMLInputElement)"
		>
			{{ (task.title ?? '').trim() }}
		</h1>
		<div class="tw:flex tw:items-center md:tw:items-stretch tw:flex-col tw:gap-1 task-properties">
			<div class="tw:flex tw:items-center tw:gap-2">
				<ColorBubble
					v-if="task.hex_color !== ''"
					:color="getHexColor(task.hex_color)"
				/>
				<BaseButton @click="copyUrl">
					<span class="title task-id">
						{{ textIdentifier }}
					</span>
				</BaseButton>
			</div>
			<span v-if="isBug" class="bug-state-pill" :class="`is-${issueStatus(task)}`">{{ ISSUE_STATUS_LABELS[issueStatus(task)] }}</span>
			<Done
				v-if="!isBug"
				:is-done="task.done"
			/>
			<BaseButton
				v-if="hasClose"
				:aria-label="$t('task.detail.closeTaskDetail')"
				class="close d-print-none"
				@click="$emit('close')"
			>
				<Icon icon="times" />
			</BaseButton>
		</div>

		<CustomTransition name="fade">
			<span
				v-if="loading && saving"
				class="is-inline-flex is-align-items-center"
			>
				<span class="loader is-inline-block mie-2" />
				{{ $t('misc.saving') }}
			</span>
			<span
				v-else-if="!loading && showSavedMessage"
				class="has-text-success is-inline-flex is-align-content-center"
			>
				<Icon
					icon="check"
					class="mie-2"
				/>
				{{ $t('misc.saved') }}
			</span>
		</CustomTransition>
	</div>
</template>

<script setup lang="ts">
import {ref, computed, onMounted, onBeforeUnmount, watch} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import {error} from '@/message'
import BaseButton from '@/components/base/BaseButton.vue'
import CustomTransition from '@/components/misc/CustomTransition.vue'
import ColorBubble from '@/components/misc/ColorBubble.vue'
import Done from '@/components/misc/Done.vue'

import {useCopyToClipboard} from '@/composables/useCopyToClipboard'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {useUpdateTaskMutation} from '@/client/queries/taskMutations'

import type {Task as ITask} from '@/client/generated'
import {getHexColor, getTaskIdentifier} from '@/helpers/task'
import {issueKind, issueStatus, ISSUE_STATUS_LABELS} from '@/helpers/issues'

const props = defineProps<{
	task: ITask,
	canWrite: boolean,
	hasClose: boolean,
}>()
const isBug = computed(() => issueKind(props.task) === 'bug')


const emit = defineEmits<{
	'update:task': [task: ITask],
	'close': [],
}>()

const router = useRouter()
const copy = useCopyToClipboard()
const {t} = useI18n({useScope: 'global'})

async function copyUrl() {
	const route = router.resolve({name: 'task.detail', query: {taskId: props.task.id}})
	const absoluteURL = new URL(route.href, window.location.href).href

	await copy(absoluteURL)
}

const updateTask = useUpdateTaskMutation()
const loading = useDelayedLoading(updateTask.isPending)

const textIdentifier = computed(() => getTaskIdentifier(props.task))

// Since loading is global state, this variable ensures we're only showing the saving icon when saving the description.
const saving = ref(false)

const showSavedMessage = ref(false)

// Track if title has unsaved changes
const titleHasChanges = ref(false)

function handleBeforeUnload(e: BeforeUnloadEvent) {
	if (titleHasChanges.value) {
		e.preventDefault()
		// Modern browsers ignore custom messages but this is still required
		e.returnValue = ''
		return ''
	}
}

onMounted(() => {
	window.addEventListener('beforeunload', handleBeforeUnload)
})

onBeforeUnmount(() => {
	window.removeEventListener('beforeunload', handleBeforeUnload)
})

// Reset titleHasChanges when the task changes
watch(() => props.task.id, () => {
	titleHasChanges.value = false
})

function handleTitleInput(event: Event) {
	const target = event.target as HTMLInputElement
	titleHasChanges.value = target.textContent !== (props.task.title ?? '')
}

async function save(element: HTMLElement) {
	const title = element.textContent ?? ''

	// An empty title would be discarded by the api, so revert and tell the user instead of failing silently.
	if (title.trim() === '') {
		element.textContent = props.task.title ?? ''
		titleHasChanges.value = false
		error({message: t('task.detail.titleRequired')})
		return
	}

	// We only want to save if the title was actually changed.
	// so we only continue if the task title changed.
	if (title === props.task.title) {
		return
	}

	try {
		saving.value = true
		const newTask = await updateTask.mutateAsync({
			...props.task,
			id: props.task.id!,
			title,
		})
		emit('update:task', newTask)
		titleHasChanges.value = false
		showSavedMessage.value = true
		setTimeout(() => {
			showSavedMessage.value = false
		}, 2000)
	} finally {
		saving.value = false
	}
}

async function cancel(element: HTMLInputElement) {
	element.textContent = props.task.title ?? ''
	titleHasChanges.value = false
	element.blur()
}
</script>

<style lang="scss" scoped>
.heading {
	display: flex;
	flex-direction: column;
	align-items: stretch;
	justify-content: flex-start;
	gap: .25rem;
	inline-size: 100%;
	padding: .25rem 0 .65rem;
	border-bottom: 1px solid var(--card-border-color);
	text-transform: none;
}
.heading h1.title {
	inline-size: 100%;
	min-block-size: 0;
	margin: 0;
	padding: .15rem .25rem;
	font-size: clamp(1.3rem, 1.8vw, 1.7rem);
	font-weight: 500;
	line-height: 1.35;
	letter-spacing: -.015em;
	text-align: start;
	border: 1px solid transparent;
	border-radius: 6px;
}
.heading h1.title[contenteditable="true"]:hover { border-color: var(--card-border-color); }
.heading h1.title[contenteditable="true"]:focus-visible {
	outline: 2px solid var(--primary);
	background: var(--white);
}
.task-properties {
	display: flex;
	flex-direction: row !important;
	align-items: center !important;
	justify-content: flex-start;
	flex-wrap: wrap;
	gap: .65rem;
	min-block-size: 1.6rem;
}
.title.task-id {font-size: .78rem; font-weight: 400; color: var(--text-light); white-space: nowrap;}
.bug-state-pill {
	display: inline-flex;
	align-items: center;
	padding: .18rem .55rem;
	font-size: .76rem;
	font-weight: 500;
	border: 1px solid var(--card-border-color);
	border-radius: 999px;
	background: var(--grey-50);
}
.bug-state-pill.is-resolved {color: #8250df;}
.bug-state-pill.is-working {color: #9a6700;}
.bug-state-pill.is-review {color: #0969da;}
.color-bubble {block-size: .75rem; inline-size: .75rem;}
.close {margin-inline-start: auto;}
@media screen and (max-width: $tablet) {
	.heading {padding-block: .35rem .65rem;}
	.heading h1.title {font-size: 1.4rem;}
}
</style>
