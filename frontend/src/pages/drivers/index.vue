<script setup lang="ts">
import DriverInspectModal from '@/components/DriverInspectModal.vue'
import { storage } from '@/wailsjs/go/models'
import * as groupStorage from '@/wailsjs/go/storage/DriverGroupStorage'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

const { t } = useI18n()

const toast = useToast()

const [route, router] = [useRoute(), useRouter()]

const groupStore = useDriverGroupStore()

const drag = ref<{ reordering: boolean; overId: number | null }>({
  reordering: false,
  overId: null
})

const modal = ref<{ inspectId: number | null; deleteId: number | null }>({
  inspectId: null,
  deleteId: null
})

async function reloadGroups() {
  return groupStorage
    .All()
    .then(gs => (groupStore.groups = gs))
    .catch(() => {
      toast.add({ title: t('toastReadDriversFailed'), color: 'error' })
    })
}

const filteredGroups = computed(() =>
  groupStore.groups.filter(g => route.query.type == undefined || g.type == route.query.type)
)

function confirmDelete() {
  if (modal.value.deleteId === null) return
  groupStorage
    .Remove(modal.value.deleteId)
    .then(() => reloadGroups())
    .catch(() => {
      toast.add({ title: t('toastDeleteFailed'), color: 'error' })
    })
    .finally(() => {
      modal.value.deleteId = null
    })
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-2">
    <PageHeader
      variant="compact"
      :title="$t('fieldInstallOption')"
      :description="$t('descInstallOption')"
    >
      <div
        class="flex flex-wrap justify-center gap-0.5 px-0.5 text-xs select-none xl:w-1/3 xl:text-sm"
      >
        <router-link
          :to="{ path: '/drivers' }"
          class="flex-1/3 truncate rounded-sm text-center font-bold uppercase shadow-xs"
          :class="{
            'bg-white text-half-baked-500': $route.query.type != undefined,
            'bg-half-baked-500 text-white': $route.query.type == undefined
          }"
          draggable="false"
        >
          {{ $t('all') }}
        </router-link>

        <router-link
          v-for="type in storage.DriverType"
          :key="type"
          :to="{ path: '/drivers', query: { type: type } }"
          class="flex-1/3 truncate rounded-sm text-center font-bold uppercase shadow-xs"
          :class="{
            'bg-white text-half-baked-500': $route.query.type !== type,
            'bg-half-baked-500 text-white': $route.query.type === type
          }"
          draggable="false"
        >
          {{ $t(`category${type.charAt(0).toUpperCase() + type.slice(1)}`) }}
        </router-link>
      </div>
    </PageHeader>

    <div
      class="scrollable flex min-h-48 grow flex-col overflow-y-scroll rounded-md p-1.5 shadow-md"
    >
      <template v-for="g in filteredGroups" :key="g.id">
        <div
          class="driver-card m-1 cursor-pointer rounded-lg border border-gray-200 px-4 py-3 shadow-sm transition-colors hover:border-half-baked-300"
          :class="{
            'select-none': drag.reordering,
            'border-half-baked-500 ring-1 ring-half-baked-500': drag.overId === g.id
          }"
          role="button"
          tabindex="0"
          :draggable="drag.reordering"
          @click="!drag.reordering && (modal.inspectId = g.id)"
          @keydown.enter.prevent="!drag.reordering && (modal.inspectId = g.id)"
          @keydown.space.prevent="!drag.reordering && (modal.inspectId = g.id)"
          @dragstart="
            event => {
              if (!drag.reordering) {
                return event.preventDefault()
              }

              event.dataTransfer!.setData('id', g.id.toString())
              const fullIdx = groupStore.groups.findIndex(g2 => g2.id === g.id)
              event.dataTransfer!.setData('index', fullIdx.toString())
            }
          "
          @dragover.prevent="drag.overId = g.id"
          @dragleave="
            event => {
              const el = event.currentTarget as HTMLElement
              if (!el.contains(event.relatedTarget as Node)) {
                drag.overId = null
              }
            }
          "
          @drop="
            event => {
              drag.overId = null

              // async function will cause event.dataTransfer to lose data
              const sourceId = parseInt(event.dataTransfer!.getData('id'))
              const sourceIdx = parseInt(event.dataTransfer!.getData('index'))
              const targetIdx = groupStore.groups.findIndex(g2 => g2.id === g.id)
              let moveBehindIdx = targetIdx
              if (sourceIdx <= targetIdx) {
                moveBehindIdx -= 1
              }
              groupStorage
                .MoveBehind(sourceId, moveBehindIdx)
                .then(() => reloadGroups())
                .catch(() => {
                  toast.add({ title: $t('toastSaveFailed'), color: 'error' })
                })
            }
          "
        >
          <div class="flex items-center justify-between gap-4">
            <div class="flex min-w-0 items-center gap-2">
              <UBadge size="sm" :style="`background-color: var(--color-${g.type})`">
                &nbsp;
              </UBadge>

              <div class="min-w-0">
                <h2 class="truncate text-base font-semibold xl:text-lg">
                  {{ g.name }}
                </h2>

                <div class="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs xl:text-sm">
                  <span class="text-gray-600">
                    {{ $t('labelDriverCount', { count: g.drivers.length }) }}
                  </span>

                  <span
                    v-if="!groupStore.isAllDriversExist(g)"
                    class="inline-flex items-center gap-0.5 rounded border border-red-200 bg-red-100 px-1.5 py-0.5 text-[10px] font-semibold text-red-600 xl:text-xs"
                    :title="
                      $t('labelPathMissing', {
                        count: g.drivers.filter(d => groupStore.notFoundDrivers.includes(d.id))
                          .length
                      })
                    "
                  >
                    <Icon icon="mdi:alert-circle" />
                    {{
                      $t('labelPathMissing', {
                        count: g.drivers.filter(d => groupStore.notFoundDrivers.includes(d.id))
                          .length
                      })
                    }}
                  </span>

                  <span
                    v-if="g.mutuallyExclusive"
                    class="inline-flex items-center gap-0.5 rounded bg-orange-100 px-1 py-0.5 text-orange-700"
                    :title="$t('fieldMutuallyExclusive')"
                  >
                    <Icon icon="mdi:chart-timeline" />
                  </span>

                  <span
                    v-if="g.drivers.some(d => d.incompatibles.length > 0)"
                    class="inline-flex items-center rounded bg-yellow-100 px-1 py-0.5 text-yellow-700"
                    :title="$t('labelIncompatibleWith')"
                  >
                    <Icon icon="mdi:source-merge" />
                  </span>

                  <span
                    v-if="g.drivers.some(d => d.allowRtCodes.length > 0)"
                    class="inline-flex items-center rounded bg-blue-100 px-1 py-0.5 text-blue-700"
                    :title="$t('fieldAllowedExitCode')"
                  >
                    <Icon icon="mdi:numeric-1-box-outline" />
                  </span>
                </div>
              </div>
            </div>

            <div class="flex shrink-0 items-center gap-1.5" @click.stop>
              <RouterLink :to="`/drivers/${g.id}/edit`" :title="$t('edit')">
                <UButton color="neutral" variant="outline" size="sm" class="h-8 w-8">
                  <Icon icon="mdi:pencil" class="text-base" />
                </UButton>
              </RouterLink>

              <UButton
                color="neutral"
                variant="outline"
                size="sm"
                class="h-8 w-8"
                :title="$t('clone')"
                @click="
                  groupStorage
                    .Clone(g.id)
                    .then(() => reloadGroups())
                    .catch(() => {
                      toast.add({ title: $t('toastSaveFailed'), color: 'error' })
                    })
                "
              >
                <Icon icon="mdi:content-duplicate" class="text-base" />
              </UButton>

              <UButton
                color="neutral"
                variant="outline"
                size="sm"
                class="h-8 w-8"
                :title="$t('delete')"
                @click="modal.deleteId = g.id"
              >
                <Icon icon="mdi:trash-can" class="text-base" />
              </UButton>
            </div>
          </div>
        </div>
      </template>

      <div
        v-if="filteredGroups.length === 0"
        class="flex flex-col items-center justify-center py-12 text-gray-400"
      >
        <Icon icon="mdi:package-variant-closed" class="mb-3 text-4xl" />

        <p class="text-sm font-medium text-gray-600 xl:text-base">{{ $t('msgNoDriverGroups') }}</p>

        <p class="mt-1 text-xs text-gray-400 xl:text-sm">{{ $t('descNoDriverGroups') }}</p>
      </div>
    </div>

    <div class="flex justify-end gap-x-3">
      <div v-show="filteredGroups.length > 1">
        <UButton
          type="button"
          size="md"
          class="text-white"
          :style="
            drag.reordering
              ? '--btn-color: var(--color-apple-green-800); animation: var(--animate-blink-75);'
              : '--btn-color: #D9BD68'
          "
          @click="drag.reordering = !drag.reordering"
        >
          {{ drag.reordering ? $t('view') : $t('fieldOrder') }}
        </UButton>
      </div>

      <RouterLink :to="{ path: '/drivers/create', query: { type: $route.query.type } }">
        <UButton color="primary" size="md">
          {{ $t('create') }}
        </UButton>
      </RouterLink>
    </div>

    <UModal
      :title="$t('confirmDeleteGroup')"
      :open="modal.deleteId !== null"
      @update:open="
        val => {
          if (!val) modal.deleteId = null
        }
      "
    >
      <template #body>
        <p>{{ $t('descDeleteGroup') }}</p>
      </template>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="modal.deleteId = null">
            {{ $t('cancel') }}
          </UButton>

          <UButton color="error" @click="confirmDelete">
            {{ $t('delete') }}
          </UButton>
        </div>
      </template>
    </UModal>

    <DriverInspectModal
      :group-id="modal.inspectId"
      @close="modal.inspectId = null"
      @edit="
        id => {
          modal.inspectId = null
          router.push(`/drivers/${id}/edit`)
        }
      "
    />
  </div>
</template>
