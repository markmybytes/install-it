import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import ChipInput from '@/components/ChipInput.vue'

// Icon is a global component registered in main.ts; stub it for standalone mounts.
const global = {
  stubs: { Icon: true }
}

type Wrapper = ReturnType<typeof mountChipInput>

function mountChipInput(props: {
  modelValue: string[]
  parse?: (raw: string) => string
  accept?: (parsed: string) => boolean
}) {
  return mount(ChipInput, {
    props,
    global
  })
}

function editInput(wrapper: Wrapper) {
  return wrapper.find('input[data-edit-input]')
}

function chipValue(wrapper: Wrapper, value: string) {
  return wrapper.get(`[data-chip-value="${value}"]`)
}

// defineModel emits 'update:modelValue' with the new array; standalone mount has
// no parent to rebind, so the emitted payload is the observable model change.
function lastModel(wrapper: Wrapper): string[] | undefined {
  const emitted = wrapper.emitted('update:modelValue')
  return emitted?.length ? (emitted[emitted.length - 1] as [string[]])[0] : undefined
}

async function openEditor(wrapper: Wrapper, value: string) {
  await chipValue(wrapper, value).trigger('click')
  await nextTick()
  return wrapper.get('input[data-edit-input]')
}

describe('outer input', () => {
  it('commits a typed value on Enter and clears the input', async () => {
    const wrapper = mountChipInput({ modelValue: ['a'] })
    const outer = wrapper.get('input[data-outer-input]')

    await outer.setValue('b')
    await outer.trigger('keydown', { key: 'Enter' })

    expect(lastModel(wrapper)).toEqual(['a', 'b'])
    expect((outer.element as HTMLInputElement).value).toBe('')
  })

  it('consumes a duplicate value rather than appending it twice', async () => {
    const wrapper = mountChipInput({ modelValue: ['a'] })
    const outer = wrapper.get('input[data-outer-input]')

    await outer.setValue('a')
    await outer.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect((outer.element as HTMLInputElement).value).toBe('')
  })
})

describe('chip editing', () => {
  it('Enter commits a valid edit at the right index', async () => {
    const wrapper = mountChipInput({ modelValue: ['a', 'b', 'c'] })

    const editor = await openEditor(wrapper, 'b')
    await editor.setValue('bb')
    await editor.trigger('keydown', { key: 'Enter' })

    expect(lastModel(wrapper)).toEqual(['a', 'bb', 'c'])
    expect(editInput(wrapper).exists()).toBe(false)
  })

  it('Enter on an accept-rejected value keeps model unchanged, editor open', async () => {
    const wrapper = mountChipInput({
      modelValue: ['1', '2'],
      accept: (parsed: string) => /^-?\d+$/.test(parsed)
    })

    const editor = await openEditor(wrapper, '1')
    await editor.setValue('x')
    await editor.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(editInput(wrapper).exists()).toBe(true)
  })

  it('Enter on a duplicate (different index) keeps model unchanged, editor open', async () => {
    const wrapper = mountChipInput({ modelValue: ['a', 'b'] })

    const editor = await openEditor(wrapper, 'a')
    await editor.setValue('b')
    await editor.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(editInput(wrapper).exists()).toBe(true)
  })

  it('Enter on an empty parsed value deletes the chip', async () => {
    const wrapper = mountChipInput({
      modelValue: ['a', 'b'],
      parse: (raw: string) => raw.trim()
    })

    const editor = await openEditor(wrapper, 'a')
    await editor.setValue('   ')
    await editor.trigger('keydown', { key: 'Enter' })

    expect(lastModel(wrapper)).toEqual(['b'])
    expect(editInput(wrapper).exists()).toBe(false)
  })

  it('Escape cancels the edit and leaves the model untouched', async () => {
    const wrapper = mountChipInput({ modelValue: ['a', 'b'] })

    const editor = await openEditor(wrapper, 'a')
    await editor.setValue('zzz')
    await editor.trigger('keydown', { key: 'Escape' })

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(editInput(wrapper).exists()).toBe(false)
  })

  it('blur commits a changed, valid edit', async () => {
    const wrapper = mountChipInput({ modelValue: ['a', 'b'] })

    const editor = await openEditor(wrapper, 'a')
    await editor.setValue('bb')
    await editor.trigger('blur')

    expect(lastModel(wrapper)).toEqual(['bb', 'b'])
  })

  it('blur reverts a changed edit that fails accept', async () => {
    const wrapper = mountChipInput({
      modelValue: ['1'],
      accept: (parsed: string) => /^-?\d+$/.test(parsed)
    })

    const editor = await openEditor(wrapper, '1')
    await editor.setValue('nope')
    await editor.trigger('blur')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(editInput(wrapper).exists()).toBe(false)
  })

  it('pins the chip width across the edit transition', async () => {
    const wrapper = mountChipInput({ modelValue: ['a', 'b', 'c'] })
    const value = chipValue(wrapper, 'b')
    const chip = value.element.parentElement as HTMLElement
    // jsdom does no layout. Give the span and the container DIFFERENT widths so
    // this test fails if the pin is taken from the clicked span instead of the
    // container — pinning the span drops padding, border, gap and the ✕.
    value.element.getBoundingClientRect = () => ({ width: 90 }) as DOMRect
    chip.getBoundingClientRect = () => ({ width: 123.5 }) as DOMRect

    await value.trigger('click')
    await nextTick()

    // Chip holds its full pre-edit width, and the editor fills the space left by
    // the ✕ rather than claiming the whole content box.
    expect(chip.style.width).toBe('123.5px')
    expect(editInput(wrapper).classes()).toContain('flex-1')
    expect(editInput(wrapper).attributes('style')).not.toContain('width')

    await editInput(wrapper).trigger('keydown', { key: 'Enter' })
    await nextTick()

    // Pin cleared on close: the chip returns to natural sizing. Unchanged text
    // keeps the same :key, so this element is reused rather than detached.
    expect(chip.style.width).toBe('')
  })

  it('focuses the editor on the first click, with the caret at the end', async () => {
    // Regression: a named template ref inside v-for does not resolve, so the
    // editor opened unfocused and needed a second click to feel "live".
    const wrapper = mount(ChipInput, {
      props: { modelValue: ['abcd'] },
      global,
      attachTo: document.body
    })

    await chipValue(wrapper, 'abcd').trigger('click')
    await nextTick()
    await nextTick()

    const input = wrapper.get('input[data-edit-input]').element as HTMLInputElement
    expect(document.activeElement).toBe(input)
    expect(input.selectionStart).toBe(4)

    wrapper.unmount()
  })

  it('keeps the value span mounted so its hover state survives an edit', async () => {
    const wrapper = mountChipInput({ modelValue: ['a'] })
    const before = chipValue(wrapper, 'a').element

    await openEditor(wrapper, 'a')

    // Same node before and after: a v-if would recreate it, and :hover does not
    // reapply to a fresh node until the mouse moves.
    expect(chipValue(wrapper, 'a').element).toBe(before)
  })

  it('keeps the delete button present while editing', async () => {
    const wrapper = mountChipInput({ modelValue: ['a'] })

    expect(wrapper.findAll('button')).toHaveLength(1)

    await openEditor(wrapper, 'a')

    // The delete button must stay in the DOM during the edit: it used to vanish
    // with the v-if that wrapped the value text.
    expect(wrapper.findAll('button')).toHaveLength(1)
  })
})
