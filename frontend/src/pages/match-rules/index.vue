<script setup lang="ts">
// import { storage } from '@/wailsjs/go/models'
import * as ruleSetStorage from '@/wailsjs/go/storage/RuleSetStorage'

const toast = useToast()

const [ruleStore, driverStore] = [useMatchRuleStore(), useDriverGroupStore()]

function hwKey(suffix: string): string {
  return `hw${suffix.charAt(0).toUpperCase() + suffix.slice(1)}`
}

function opKey(suffix: string): string {
  return `op${suffix.charAt(0).toUpperCase() + suffix.slice(1)}`
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-2">
    <PageHeader
      variant="compact"
      :title="$t('titleMatchRule')"
      :description="$t('descMatchRule')"
    />

    <div
      v-scroll-restore="'match-rules-list'"
      class="flex min-h-48 grow flex-col overflow-y-scroll rounded-md p-1.5 shadow-md"
    >
      <div
        v-for="rs in ruleStore.ruleSets"
        :key="rs.id"
        class="driver-card m-1 rounded-lg border border-gray-200 px-2 py-1 shadow-sm"
      >
        <div class="flex justify-between">
          <div class="flex min-w-0 items-center gap-1">
            <UBadge v-if="rs.should_hit_all" size="sm" class="bg-rose-400 text-nowrap text-white">
              {{ $t('fieldHitAll') }}
            </UBadge>

            <h2 class="truncate">{{ rs.name }}</h2>
          </div>

          <div class="flex gap-x-1.5 py-1">
            <RouterLink :to="`/match-rules/${rs.id}/edit`" :title="$t('edit')">
              <UButton color="neutral" variant="outline" size="xs" class="h-6">
                <Icon icon="mdi:pencil" class="text-gray-500" />
              </UButton>
            </RouterLink>

            <UButton
              color="neutral"
              variant="outline"
              size="xs"
              class="h-6"
              :title="$t('clone')"
              @click="
                ruleSetStorage.Clone(rs.id).then(() =>
                  ruleSetStorage
                    .All()
                    .then(rs => (ruleStore.ruleSets = rs))
                    .catch(() => {
                      toast.add({ title: $t('toastReadDriversFailed'), color: 'error' })
                    })
                )
              "
            >
              <Icon icon="mdi:content-duplicate" class="text-gray-500" />
            </UButton>

            <UButton
              color="neutral"
              variant="outline"
              size="xs"
              class="h-6"
              :title="$t('delete')"
              @click="
                ruleSetStorage.Remove(rs.id).then(() =>
                  ruleSetStorage
                    .All()
                    .then(rs => (ruleStore.ruleSets = rs))
                    .catch(() => {
                      toast.add({ title: $t('toastReadDriversFailed'), color: 'error' })
                    })
                )
              "
            >
              <Icon icon="mdi:trash-can" class="text-gray-500" />
            </UButton>
          </div>
        </div>

        <div class="grid grid-cols-10 gap-1 bg-gray-100 py-1 text-xs">
          <div class="col-span-2 font-semibold">{{ $t('fieldSource') }}</div>

          <div class="col-span-2 font-semibold">{{ $t('fieldOperator') }}</div>

          <div class="col-span-6 font-semibold">{{ $t('fieldPattern') }}</div>
        </div>

        <div
          v-for="(r, ri) in rs.rules"
          :key="ri"
          class="grid grid-cols-10 items-center gap-1 py-1 text-xs"
        >
          <div class="col-span-2">
            {{ $t(hwKey(r.source)) }}
          </div>

          <div class="col-span-2">
            <span class="font-mono">
              {{ $t(opKey(r.operator)) }}
            </span>
          </div>

          <div class="col-span-6">
            <div class="line-clamp-2">
              <UBadge
                v-if="r.should_hit_all"
                size="sm"
                class="me-0.5 bg-rose-400 text-white md:me-1"
              >
                {{ $t('fieldHitAll') }}
              </UBadge>

              <UBadge
                v-if="r.is_case_sensitive"
                size="sm"
                class="me-0.5 bg-orange-300 text-white md:me-1"
              >
                Aa
              </UBadge>

              <UBadge
                v-for="(v, i) in r.values"
                :key="i"
                size="sm"
                color="tertiary"
                class="me-0.5 md:me-1"
              >
                {{ v }}
              </UBadge>
            </div>
          </div>
        </div>

        <hr class="my-1" />

        <div class="flex gap-2 text-xs">
          <p class="content-center font-semibold">{{ $t('fieldMatchTo') }}</p>

          <div class="line-clamp-2 flex-1">
            <UBadge
              v-for="(group, i) in driverStore.groups.filter(g =>
                rs.driver_group_ids?.includes(g.id)
              )"
              :key="i"
              size="sm"
              class="me-0.5 text-zinc-600"
              :style="`background-color: var(--color-${group.type})`"
            >
              {{ group.name }}
            </UBadge>
          </div>
        </div>
      </div>
    </div>

    <div class="flex justify-end gap-x-3">
      <RouterLink :to="{ path: '/match-rules/create' }">
        <UButton color="primary" size="sm">
          {{ $t('create') }}
        </UButton>
      </RouterLink>
    </div>
  </div>
</template>
