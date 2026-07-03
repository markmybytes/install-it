<script setup lang="ts">
import DriverEditor from '@/components/DriverEditor.vue'
import { storage } from '@/wailsjs/go/models'
import * as groupStorage from '@/wailsjs/go/storage/DriverGroupStorage'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

const props = defineProps<{ id?: number }>()

const { t } = useI18n()

const [route, router] = [useRoute(), useRouter()]

const toast = useToast()

const groupStore = useDriverGroupStore()

const sourceGroup = computed(
  () =>
    groupStore.groups.find(g => g.id === props.id) ??
    new storage.DriverGroup({
      type:
        storage.DriverType[
          (route.query.type as string | undefined)?.toUpperCase() as keyof typeof storage.DriverType
        ] ?? storage.DriverType.NETWORK,
      name: '',
      drivers: []
    })
)

const { data: group, reset } = useEditor({
  source: sourceGroup,
  warnOnUnsavedLeave: true
})

const ui = ref<{
  expanded: Set<number>
  nextTempId: number
}>({
  expanded: new Set(),
  nextTempId: -1
})

function addDriver() {
  const id = ui.value.nextTempId
  ui.value.nextTempId -= 1
  group.value.drivers.push(
    new storage.Driver({
      id,
      type: group.value.type,
      name: '',
      path: '',
      flags: [],
      minExeTime: 5,
      allowRtCodes: [],
      incompatibles: []
    })
  )
  ui.value.expanded.add(id)
  ui.value.expanded = new Set(ui.value.expanded)
}

function removeDriver(id: number) {
  group.value.drivers = group.value.drivers.filter(d => d.id !== id)
  ui.value.expanded.delete(id)
  ui.value.expanded = new Set(ui.value.expanded)
}

function toggleDriver(id: number) {
  const next = new Set(ui.value.expanded)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  ui.value.expanded = next
}

const allExpanded = computed(() => {
  if (group.value.drivers.length === 0) return false
  return group.value.drivers.every(d => ui.value.expanded.has(d.id))
})

function toggleAll() {
  if (allExpanded.value) {
    ui.value.expanded = new Set()
  } else {
    ui.value.expanded = new Set(group.value.drivers.map(d => d.id))
  }
}

function handleSubmit() {
  if (!group.value.name?.trim()) {
    toast.add({ title: t('toastGroupNameRequired'), color: 'warning' })
    return
  }

  if (group.value.drivers.length == 0) {
    toast.add({ title: t('toastAddDriverRequired'), color: 'warning' })
    return
  }

  if (group.value.drivers.some(d => !d.path?.trim())) {
    toast.add({ title: t('toastPathRequired'), color: 'warning' })
    return
  }

  const handleSuccess = () => {
    toast.add({ title: t('toastUpdated'), color: 'success' })
    groupStorage
      .All()
      .then(newDriverGroups => {
        groupStore.groups = newDriverGroups
        return reset()
      })
      .then(() => {
        router.back()
      })
  }

  if (!group.value.id) {
    groupStorage
      .Add(group.value)
      .then(handleSuccess)
      .catch(reason => toast.add({ title: reason.toString(), color: 'error' }))
  } else {
    groupStorage
      .Update(group.value)
      .then(handleSuccess)
      .catch(reason => toast.add({ title: reason.toString(), color: 'error' }))
  }
}
</script>

<template>
  <form
    class="mx-auto flex h-full max-w-full flex-col gap-y-5 overflow-y-scroll lg:max-w-2xl xl:max-w-4xl"
    autocomplete="off"
    @submit.prevent="handleSubmit"
  >
    <div class="flex gap-4">
      <div class="w-48 shrink-0">
        <label class="mb-1 block text-xs font-bold tracking-wider text-gray-500 uppercase">
          {{ $t('fieldDriverType') }} <span class="text-red-500">*</span>
        </label>

        <USelect
          v-model="group.type"
          name="type"
          class="w-full"
          :items="
            Object.values(storage.DriverType).map(type => ({
              label: $t(`category${type.charAt(0).toUpperCase() + type.slice(1)}`),
              value: type
            }))
          "
          required
        />
      </div>

      <div class="flex-1">
        <label class="mb-1 block text-xs font-bold tracking-wider text-gray-500 uppercase">
          {{ $t('name') }} <span class="text-red-500">*</span>
        </label>

        <UInput v-model="group.name" type="text" class="w-full" required />
      </div>
    </div>

    <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <label class="flex cursor-pointer items-start gap-3">
        <UCheckbox v-model="group.mutuallyExclusive" class="mt-1" />

        <div>
          <span class="block text-xs font-bold text-gray-800 xl:text-sm">{{
            $t('fieldMutuallyExclusive')
          }}</span>

          <p class="mt-0.5 text-xs text-gray-500">{{ $t('descMutuallyExclusive') }}</p>
        </div>
      </label>
    </div>

    <div class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-2">
        <label class="block text-xs font-bold tracking-wider text-gray-500 uppercase">
          {{ $t('fieldDriver') }}
        </label>

        <div class="flex items-center gap-2">
          <UButton
            v-if="group.drivers.length > 0"
            type="button"
            color="neutral"
            variant="outline"
            size="sm"
            @click="toggleAll"
          >
            <Icon
              :icon="allExpanded ? 'mdi:unfold-less-horizontal' : 'mdi:unfold-more-horizontal'"
              class="mr-1 text-xs xl:text-sm"
            />
            {{ allExpanded ? $t('labelCollapseAll') : $t('labelExpandAll') }}
          </UButton>

          <UButton type="button" color="primary" size="sm" @click="addDriver">
            <Icon icon="mdi:plus-circle-outline" class="mr-1 text-xs xl:text-sm" />
            {{ $t('labelAddDriver') }}
          </UButton>
        </div>
      </div>

      <div
        v-if="group.drivers.length === 0"
        class="flex flex-col items-center rounded-lg border border-dashed border-gray-300 bg-white py-10 text-center text-gray-400"
      >
        <Icon icon="mdi:package-variant" class="mb-2 text-4xl opacity-50" />

        <span class="text-xs font-medium xl:text-sm">{{ $t('msgNoDriversInGroup') }}</span>
      </div>

      <DriverEditor
        v-for="(d, i) in group.drivers"
        :key="d.id"
        v-model:driver="group.drivers[i]!"
        :index="i"
        :is-new="d.id < 0"
        :expanded="ui.expanded.has(d.id)"
        @remove="removeDriver"
        @toggle="toggleDriver"
      />
    </div>

    <div class="mt-4 flex shrink-0 gap-4 border-t border-gray-200 pt-4">
      <UButton
        type="button"
        color="neutral"
        variant="outline"
        class="flex-1 justify-center text-sm"
        @click="$router.back()"
      >
        {{ $t('back') }}
      </UButton>

      <UButton type="submit" color="secondary" class="flex-1 justify-center text-sm">
        {{ $t('save') }}
      </UButton>
    </div>
  </form>
</template>
