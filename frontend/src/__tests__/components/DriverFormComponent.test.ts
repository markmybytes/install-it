import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createI18n } from 'vue-i18n'
import { createRouter, createMemoryHistory } from 'vue-router'
import DriverFormComponent from '@/components/DriverFormComponent.vue'
import { useEditor } from '@/composables/useEditor'
import * as groupStorage from '@/wailsjs/go/storage/DriverGroupStorage'
import { storage } from '@/wailsjs/go/models'

// UInput is not registered in vitest, so v-model cannot be exercised through the
// inert custom element. Register a minimal real stub that proxies to a native input.
const UInputStub = defineComponent({
  name: 'UInput',
  props: { modelValue: { type: [String, Number], default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value)
      })
  }
})

// `stubs: { UButton: true }` renders an empty <u-button-stub> with no slot content,
// so the add-driver icon would be unreachable. Render a real button instead so the
// click can be located by its contained icon.
const UButtonStub = defineComponent({
  name: 'UButton',
  inheritAttrs: false,
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.())
  }
})

const toastAdd = vi.fn()

let fakeStore: { groups: storage.DriverGroup[] }

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }]
  })
}

function makeI18n() {
  return createI18n({
    legacy: false,
    locale: 'en',
    missingWarn: false,
    fallbackWarn: false,
    messages: {}
  })
}

async function mountForm(props: { id?: number } = {}) {
  const router = makeRouter()
  // Two entries so router.back() has somewhere to go after submit.
  await router.push('/?x=1')
  await router.push('/')
  const wrapper = mount(DriverFormComponent, {
    props,
    global: {
      stubs: {
        UInput: UInputStub,
        USelect: true,
        UButton: UButtonStub,
        UCheckbox: true,
        DriverSelector: true,
        ChipInput: true
      },
      plugins: [makeI18n(), router]
    }
  })
  return { wrapper, router }
}

function findAddButton(wrapper: ReturnType<typeof mount>) {
  return wrapper
    .findAll('button')
    .find(b => b.find('icon[icon="mdi:plus-circle-outline"]').exists())
}

describe('DriverFormComponent', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fakeStore = { groups: [] }
    vi.stubGlobal('useToast', () => ({ add: toastAdd }))
    vi.stubGlobal('useDriverGroupStore', () => fakeStore)
    vi.stubGlobal('useEditor', useEditor)
    vi.stubGlobal('useUnsavedFormStore', () => ({ show: false, setAnswerHandler: vi.fn() }))
    vi.mocked(groupStorage.Add).mockResolvedValue(undefined)
    vi.mocked(groupStorage.Update).mockResolvedValue(undefined)
    vi.mocked(groupStorage.All).mockResolvedValue([])
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends id 0 (never negative) for a newly-added driver', async () => {
    const { wrapper } = await mountForm()

    const addButton = findAddButton(wrapper)
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')

    await wrapper.find('input').setValue('G1')

    const editor = wrapper.findComponent({ name: 'DriverEditor' })
    const driver = editor.props('driver') as storage.Driver
    driver.path = 'C:/net.exe'

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(vi.mocked(groupStorage.Add)).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(groupStorage.Add).mock.calls[0]![0]
    expect(payload.drivers[0]!.id).toBe(0)
    expect(payload.drivers[0]!.id).toBeGreaterThanOrEqual(0)
    expect(vi.mocked(groupStorage.Update)).not.toHaveBeenCalled()
    expect(toastAdd).toHaveBeenCalledWith(expect.objectContaining({ color: 'success' }))
  })

  it('preserves real ids when updating an existing group', async () => {
    fakeStore.groups = [
      new storage.DriverGroup({
        id: 5,
        name: 'G',
        type: storage.DriverType.NETWORK,
        mutuallyExclusive: false,
        drivers: [
          new storage.Driver({
            id: 1,
            type: storage.DriverType.NETWORK,
            name: 'D1',
            path: 'C:/net.exe',
            flags: [],
            minExeTime: 5,
            allowRtCodes: [],
            incompatibles: []
          })
        ]
      })
    ]

    const { wrapper } = await mountForm({ id: 5 })

    const editor = wrapper.findComponent({ name: 'DriverEditor' })
    const driver = editor.props('driver') as storage.Driver
    driver.path = 'C:/other.exe'

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(vi.mocked(groupStorage.Update)).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(groupStorage.Update).mock.calls[0]![0]
    expect(payload.drivers[0]!.id).toBe(1)
    expect(vi.mocked(groupStorage.Add)).not.toHaveBeenCalled()
  })
})
