<script setup lang="ts">
import { nextTick, ref, useTemplateRef } from 'vue'

const props = withDefaults(
  defineProps<{
    placeholder?: string
    commitKeys?: Array<'enter' | ' ' | 'tab' | 'comma'>
    parse?: (raw: string) => string
    accept?: (parsed: string) => boolean
    popOnBackspace?: boolean
    minInputWidth?: string
  }>(),
  {
    placeholder: '',
    commitKeys: () => ['enter'],
    parse: undefined,
    accept: undefined,
    popOnBackspace: true,
    minInputWidth: '120px'
  }
)

const model = defineModel<Array<string>>({ required: true })

const input = ref('')

// One editor at a time. Single ref so opening is one atomic assignment: the
// swap and the width pin must land together. `pin` holds the chip's measured
// width so the flex-wrap row never reflows mid-edit.
const edit = ref({
  index: null as number | null,
  text: '',
  invalid: false,
  pin: null as string | null
})

const outerInput = useTemplateRef<HTMLInputElement>('outerInput')

// Function ref, not useTemplateRef: this input lives inside a v-for, so a
// named template ref does not resolve to it. Guard also rejects the Comment
// placeholder Vue binds while the v-else branch is inactive.
let editInput: HTMLInputElement | null = null
const setEditRef = (el: unknown) => {
  editInput = el instanceof HTMLInputElement ? el : null
}

const parseValue = (raw: string) => (props.parse ?? ((s: string) => s))(raw)

// Shared by the outer input and the chip editor. editIndex excludes the chip
// being edited from duplicate detection.
function tryCommit(raw: string, editIndex: number | null = null): 'ok' | 'invalid' | 'duplicate' {
  const parsed = parseValue(raw)
  if (!parsed || !(props.accept?.(parsed) ?? true)) {
    return 'invalid'
  }
  if (model.value.some((value, idx) => value === parsed && idx !== editIndex)) {
    return 'duplicate'
  }
  model.value =
    editIndex === null
      ? [...model.value, parsed]
      : model.value.map((value, idx) => (idx === editIndex ? parsed : value))
  return 'ok'
}

function onOuterKeydown(event: KeyboardEvent) {
  if (!props.commitKeys.includes(event.key.toLowerCase() as 'enter' | ' ' | 'tab' | 'comma')) {
    return
  }
  event.preventDefault()

  // A duplicate still clears the input; only 'invalid' keeps it.
  if (tryCommit(input.value) === 'invalid') {
    return
  }
  input.value = ''
}

function startEdit(i: number, event: Event) {
  onEditBlurFinalize()
  // Pin the chip CONTAINER, not the clicked span: the span is only the text
  // box, so its width excludes the container's padding, border, gap and ✕ —
  // pinning that would shrink the chip on every edit.
  const width = (event.currentTarget as HTMLElement).parentElement!.getBoundingClientRect().width
  edit.value = {
    index: i,
    text: model.value[i] ?? '',
    invalid: false,
    // Zero width (jsdom, hidden chip) → fall back to ch sizing.
    pin: width > 0 ? `${width}px` : null
  }
  nextTick(() => {
    if (!editInput) return
    editInput.focus()
    // Caret at end, not select() — selecting everything renders as a highlight.
    editInput.setSelectionRange(editInput.value.length, editInput.value.length)
  })
}

function onEditKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeEditor(true)
    return
  }
  if (event.key !== 'Enter') {
    return
  }
  event.preventDefault()

  const i = edit.value.index
  if (i === null) {
    return
  }

  // Must match tryCommit's own parse of the same string.
  if (parseValue(edit.value.text) === '') {
    model.value = model.value.filter((_, idx) => idx !== i)
    closeEditor(true)
    return
  }
  if (tryCommit(edit.value.text, i) === 'ok') {
    closeEditor(true)
  } else {
    edit.value.invalid = true
  }
}

// Blur and chip-switch land here: unchanged → cancel, changed → commit when
// valid else revert. Never strands a half-open editor.
function onEditBlurFinalize() {
  const i = edit.value.index
  if (i === null) {
    return
  }
  if (edit.value.text !== model.value[i]) {
    tryCommit(edit.value.text, i)
  }
  closeEditor(false)
}

function closeEditor(refocus: boolean) {
  edit.value = { index: null, text: '', invalid: false, pin: null }
  if (refocus) {
    nextTick(() => outerInput.value?.focus())
  }
}
</script>

<template>
  <div
    class="flex flex-wrap items-center gap-1.5 rounded-lg border border-gray-200 bg-gray-50/50 px-2 py-1.5 shadow-inner transition-all focus-within:bg-white"
    :style="{
      '--chip-bg': 'var(--color-half-baked-50)',
      '--chip-border': 'var(--color-half-baked-200)',
      '--chip-text': 'var(--color-half-baked-700)',
      '--chip-close-text': 'var(--color-half-baked-400)',
      '--chip-close-bg-hover': 'var(--color-half-baked-100)',
      '--focus-border': 'var(--color-half-baked-500)',
      '--focus-ring': 'color-mix(in oklab, var(--color-half-baked-500) 30%, transparent)'
    }"
  >
    <span
      v-for="(item, i) in model"
      :key="`${item}-${i}`"
      class="inline-flex max-w-full min-w-0 items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-bold shadow-sm"
      :class="edit.invalid && edit.index === i ? 'ring-1 ring-red-400' : ''"
      :style="{
        backgroundColor: 'var(--chip-bg)',
        borderColor: 'var(--chip-border)',
        color: 'var(--chip-text)',
        width: edit.index === i && edit.pin ? edit.pin : undefined
      }"
    >
      <!-- v-show, not v-if: destroying this span on edit drops its :hover state and
           it only comes back once the mouse moves. -->
      <span
        v-show="edit.index !== i"
        :data-chip-value="item"
        class="min-w-0 cursor-text truncate font-mono transition-colors hover:underline hover:decoration-dotted hover:underline-offset-2"
        @click="startEdit(i, $event)"
        >{{ item }}</span
      >

      <input
        v-if="edit.index === i"
        :ref="setEditRef"
        v-model="edit.text"
        data-edit-input
        type="text"
        class="min-w-0 border-none bg-transparent px-0 py-0 font-mono text-xs font-bold focus:ring-0 focus:outline-none"
        :class="edit.pin ? 'flex-1' : ''"
        :style="{
          color: 'inherit',
          width: edit.pin ? undefined : `min(${Math.max(edit.text.length, 2) + 1}ch, 40ch)`
        }"
        @keydown="onEditKeydown"
        @input="edit.invalid && (edit.invalid = false)"
        @blur="onEditBlurFinalize"
      />

      <!-- Rendered in both states: the ✕ must never vanish from a chip. -->
      <button
        type="button"
        class="rounded-sm p-0.5 transition-colors"
        :style="{
          color: 'var(--chip-close-text)',
          '--hover-bg': 'var(--chip-close-bg-hover)',
          '--hover-text': 'var(--chip-text)'
        }"
        @click="model = model.filter((_, idx) => idx !== i)"
      >
        <Icon icon="mdi:close" class="h-3 w-3" />
      </button>
    </span>

    <input
      ref="outerInput"
      v-model="input"
      data-outer-input
      type="text"
      :placeholder="placeholder"
      class="border-none bg-transparent px-2 py-1 font-mono text-xs text-gray-800 placeholder:text-gray-400 focus:ring-0 focus:outline-none xl:text-sm"
      :style="{ minWidth: minInputWidth, flex: '1 1 0%' }"
      @keydown="onOuterKeydown"
      @keydown.backspace="
        popOnBackspace && input === '' && model.length > 0 && (model = model.slice(0, -1))
      "
    />

    <slot />
  </div>
</template>

<style scoped>
div:focus-within {
  border-color: var(--focus-border);
  box-shadow: 0 0 0 2px var(--focus-ring);
}

button:hover,
button:focus-visible {
  background-color: var(--hover-bg);
  color: var(--hover-text);
}
</style>
