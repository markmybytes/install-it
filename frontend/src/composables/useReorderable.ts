import Sortable from 'sortablejs'

/**
 * Manages SortableJS drag-and-drop reorder mode.
 *
 * @param el - Template ref name (string) or pre-built `Ref<HTMLElement | null>`.
 * @param options - Partial `Sortable.Options` merged with defaults.
 * @returns `{ enabled, container, sortable }` — toggle, container ref,
 *   and the Sortable instance.
 *
 * @example
 * const { enabled } = useReorderable('list', { onEnd: handler })
 * // <div ref="list">
 */
export function useReorderable(
  el: string | Readonly<Ref<HTMLElement | null>>,
  options: Partial<Sortable.Options> = {}
): {
  enabled: Ref<boolean>
  sortable: Readonly<Ref<Sortable | null>>
  container: Readonly<Ref<HTMLElement | null>>
} {
  const container: Readonly<Ref<HTMLElement | null>> = isRef(el)
    ? el
    : useTemplateRef<HTMLElement>(el)
  const enabled = ref(false)
  const sortable = shallowRef<Sortable | null>(null)

  const merged: Sortable.Options = {
    handle: '.drag-handle',
    animation: 200,
    easing: 'cubic-bezier(0.25, 1, 0.5, 1)',
    ghostClass: 'sortable-ghost',
    dragClass: 'sortable-drag',
    ...options
  }

  onMounted(() => {
    if (!container.value) {
      return
    }
    sortable.value = new Sortable(container.value, { ...merged, disabled: !enabled.value })
  })

  watch(enabled, val => {
    sortable.value?.option('disabled', !val)
  })

  onBeforeUnmount(() => {
    sortable.value?.destroy()
    sortable.value = null
  })

  return { enabled, sortable, container }
}
