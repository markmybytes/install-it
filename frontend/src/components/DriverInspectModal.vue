<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  groupId: number | null
}>()

const emit = defineEmits<{
  close: []
  edit: [id: number]
}>()

const groupStore = useDriverGroupStore()

const group = computed(() => groupStore.groups.find(g => g.id === props.groupId))

const isOpen = ref(props.groupId !== null)

// v-model:open + @after:leave (not @update:open) keeps groupId non-null during
// the leave animation, preventing "Group Not Found" from flashing on close.
watch(
  () => props.groupId,
  val => {
    isOpen.value = val !== null
  }
)
</script>

<template>
  <UModal v-model:open="isOpen" :title="$t('titleInspectGroup')" @after:leave="emit('close')">
    <template #body>
      <div v-if="group" class="flex flex-col gap-y-4">
        <div
          class="grid grid-cols-3 gap-2 rounded-lg border border-gray-200 bg-white p-3 text-center shadow-sm"
        >
          <div>
            <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
              $t('fieldDriverType')
            }}</span>

            <UBadge
              size="sm"
              class="mt-0.5 text-zinc-600"
              :style="`background-color: var(--color-${group.type})`"
            >
              {{ $t(`category${group.type.charAt(0).toUpperCase() + group.type.slice(1)}`) }}
            </UBadge>
          </div>

          <div class="border-x border-gray-100">
            <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">
              {{ $t('fieldExclusiveFlow') }}
            </span>

            <span class="text-sm font-bold text-gray-800 xl:text-sm">
              {{ group.mutuallyExclusive ? $t('labelYes') : $t('labelNo') }}
            </span>
          </div>

          <div>
            <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">
              {{ $t('fieldDriver') }}
            </span>

            <span class="text-sm font-bold text-gray-800 xl:text-sm">{{
              group.drivers.length
            }}</span>
          </div>
        </div>

        <!-- Per-driver cards -->
        <div
          v-for="(d, i) in group.drivers"
          :key="d.id"
          class="relative space-y-3 overflow-hidden rounded-lg border border-gray-200 bg-white p-4 shadow-sm"
        >
          <!-- Left edge accent -->
          <div
            class="absolute top-0 left-0 h-full w-1"
            :class="groupStore.notFoundDrivers.includes(d.id) ? 'bg-red-400' : 'bg-primary-200'"
          ></div>

          <!-- Header -->
          <div class="flex items-center justify-between pl-2">
            <h4
              class="flex items-center gap-2 text-base font-bold xl:text-base"
              :class="d.name ? 'text-gray-900' : 'text-gray-400'"
            >
              <span class="text-gray-400">#{{ i + 1 }}</span>
              {{ d.name || $t('msgUnnamedDriver') }}
            </h4>
            <!-- Only show badge when path is MISSING, not when valid -->
            <span
              v-if="groupStore.notFoundDrivers.includes(d.id)"
              class="rounded border border-red-200 bg-red-100 px-2 py-0.5 text-xs font-bold text-red-700 xl:text-xs"
            >
              {{ $t('labelMissingExe') }}
            </span>
          </div>

          <!-- Path -->
          <div class="space-y-2 pl-2">
            <div>
              <span class="text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
                $t('path')
              }}</span>

              <div
                class="rounded border border-gray-100 bg-gray-50 p-2 font-mono text-xs break-all text-gray-700 xl:text-sm"
              >
                {{ d.path }}
              </div>
            </div>

            <!-- Details grid -->
            <div class="grid grid-cols-2 gap-3 rounded border border-gray-100 bg-gray-50/50 p-2">
              <div>
                <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
                  $t('fieldArgument')
                }}</span>

                <span class="mt-0.5 block font-mono text-xs font-bold text-gray-700 xl:text-sm">
                  {{ d.flags.length > 0 ? d.flags.join(' ') : '--' }}
                </span>
              </div>

              <div>
                <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
                  $t('fieldAllowedExitCode')
                }}</span>

                <span class="mt-0.5 block font-mono text-xs font-bold text-gray-700 xl:text-sm">
                  {{ d.allowRtCodes.length > 0 ? d.allowRtCodes.join(', ') : '--' }}
                </span>
              </div>

              <div>
                <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
                  $t('fieldMinExecuteTime')
                }}</span>

                <span class="mt-0.5 block font-mono text-xs font-bold text-gray-700 xl:text-sm">{{
                  d.minExeTime > 0 ? d.minExeTime + 's' : '--'
                }}</span>
              </div>

              <div>
                <span class="block text-xs font-bold text-gray-400 uppercase xl:text-xs">{{
                  $t('labelIncompatibleWith')
                }}</span>

                <span class="mt-0.5 block font-mono text-xs font-bold text-gray-700 xl:text-sm">
                  {{ d.incompatibles.length > 0 ? d.incompatibles.length : '--' }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="py-6 text-center text-base text-gray-500 xl:text-base">
        {{ $t('msgGroupNotFound') }}
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton color="neutral" variant="ghost" size="sm" @click="isOpen = false">
          {{ $t('cancel') }}
        </UButton>

        <UButton v-if="group" color="primary" size="sm" @click="emit('edit', group.id)">
          {{ $t('edit') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
