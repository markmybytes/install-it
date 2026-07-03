<script setup lang="ts">
import type { storage } from '@/wailsjs/go/models'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  driverGroups: Array<storage.DriverGroup>
  excludes?: Array<number>
  groupBy: 'group' | 'driver'
}>()

const model = defineModel<Array<number> | undefined>({ default: [] })

const { t: $t } = useI18n()

const searchPhrase = ref('')

// ensure model is always an array
watch(
  () => model.value,
  val => {
    if (val === null || val === undefined) {
      model.value = []
    }
  },
  { immediate: true }
)

const filteredGroups = computed(() => {
  return searchPhrase.value === ''
    ? props.driverGroups
    : props.driverGroups.filter(
        g =>
          g.name.includes(searchPhrase.value) ||
          g.drivers.some(d => d.name.includes(searchPhrase.value))
      )
})

const groupItems = computed(() =>
  props.groupBy === 'group'
    ? filteredGroups.value.map(g => ({ label: g.name, value: g.id, type: g.type }))
    : filteredGroups.value.flatMap(g =>
        g.drivers
          .filter(d => !props.excludes?.includes(d.id))
          .map(d => ({ label: `[${g.name}] ${d.name}`, value: d.id, groupType: g.type }))
      )
)
</script>

<template>
  <div>
    <div class="mb-1 line-clamp-1 text-sm xl:text-base">
      <span class="inline">
        {{ $t('labelSelectedCount', { count: model?.length ?? 0 }) }}
      </span>
    </div>

    <div class="mb-2 flex gap-x-3">
      <UInput
        v-model="searchPhrase"
        :placeholder="$t('search')"
        class="ms-1 grow"
        variant="none"
        :ui="{ base: 'border-none bg-gray-100 focus:outline-gray-200' }"
      />

      <UButton
        type="button"
        class="px-2 text-white"
        style="--btn-color: var(--color-powder-blue-800)"
        :title="$t('labelSelectAll')"
        @click="
          () => {
            model =
              $props.groupBy === 'group'
                ? props.driverGroups.map(g => g.id)
                : props.driverGroups.flatMap(g => g.drivers.map(d => d.id))
          }
        "
      >
        <Icon icon="mdi:checkbox-marked" />
      </UButton>

      <UButton
        type="button"
        class="px-2 text-white"
        style="--btn-color: var(--color-rose-400)"
        :title="$t('labelSelectNone')"
        @click="model = []"
      >
        <Icon icon="mdi:checkbox-blank-outline" />
      </UButton>
    </div>

    <div class="rounded-lg border p-2">
      <div class="h-44 space-y-2 overflow-y-auto">
        <!-- "No matches" empty state -->
        <div
          v-if="groupItems.length === 0 && searchPhrase"
          class="py-4 text-center text-sm text-gray-400 italic xl:text-base"
        >
          {{ $t('msgNoMatches') }}
        </div>

        <!-- Group/Driver items with color blocks -->
        <template v-for="item in groupItems" :key="item.value">
          <label class="flex cursor-pointer items-center select-none">
            <UCheckbox
              size="sm"
              :model-value="model?.includes(item.value) ?? false"
              @update:model-value="
                (checked: boolean | 'indeterminate') => {
                  if (checked === true) {
                    model = [...(model || []), item.value]
                  } else {
                    model = model?.filter(v => v !== item.value) ?? []
                  }
                }
              "
            />

            <UBadge
              v-if="'type' in item ? item.type : 'groupType' in item ? item.groupType : undefined"
              size="sm"
              class="ms-1.5"
              :style="`background-color: var(--color-${'type' in item ? item.type : item.groupType})`"
            >
              &nbsp;
            </UBadge>

            <span class="ms-1.5 text-sm xl:text-base">{{ item.label }}</span>
          </label>
        </template>
      </div>
    </div>
  </div>
</template>
