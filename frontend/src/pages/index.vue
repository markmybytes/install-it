<script setup lang="ts">
import CommandStatueModal from '@/components/CommandStatusModal.vue'
import { type Command } from '@/types/execute'
import * as utils from '@/utils'
import * as executor from '@/wailsjs/go/execute/CommandExecutor'
import * as matcher from '@/wailsjs/go/matching/Matcher'
import { sysinfo } from '@/wailsjs/go/models'
import * as sysinfoApi from '@/wailsjs/go/sysinfo/SysInfo'
import { computed, onBeforeMount, onBeforeUnmount, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { decodeError } from '@/utils/index'

const { t } = useI18n()

const toast = useToast()

function hwKey(part: string): string {
  return `hw${part.charAt(0).toUpperCase() + part.slice(1)}`
}

const [statusModal, form] = [useTemplateRef('statusModal'), useTemplateRef('form')]

const [groupStore, settingStore] = [useDriverGroupStore(), useAppSettingStore()]

/** Driver groups with conditional exclusion based on app configuration */
const groups = computed(() =>
  settingStore.settings.hide_not_found
    ? groupStore.groups.filter(g => groupStore.isAllDriversExist(g))
    : groupStore.groups
)

const systemInfo = ref<{ hw: sysinfo.ResolvedHardware | null; os: sysinfo.OSInfo | null }>({
  hw: null,
  os: null
})

const cpuTemp = ref<number | null>(null)
const cpuTempError = ref<string | null>(null)
const tempToasted = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

const selectedNetwork = ref<number>(0)
const selectedDisplay = ref<number>(0)
const selectedMiscellaneous = ref<number[]>([])

function loadSystemInfo() {
  Promise.all([utils.getHardware(), utils.getOSInfo()]).then(([hw, os]) => {
    systemInfo.value = { hw, os }
  })
}

onBeforeMount(() => {
  loadSystemInfo()

  if (settingStore.settings.enable_cpu_temp) {
    const tick = () =>
      sysinfoApi
        .CPUTemperature()
        .then(t => {
          cpuTempError.value = null
          cpuTemp.value = Math.round(t)
        })
        .catch(err => {
          cpuTempError.value = decodeError(err, t)
          if (!tempToasted.value) {
            // Color = 'warning' for warn* codes (per handoff §3.3 prefix guidance)
            const color = String(err?.code ?? '').startsWith('warn') ? 'warning' : 'error'
            toast.add({ title: decodeError(err, t), color })
            tempToasted.value = true
          }
          // stop polling on permanent failure: timer = null (only if code is permanent; for now leave polling alone)
        })
        .finally(() => {
          if (timer === null) return
          const secs = Math.max(1, Number(settingStore.settings.cpu_temp_refresh_interval) || 5)
          timer = setTimeout(tick, secs * 1000)
        })
    timer = setTimeout(tick, 0)
  }
})

function selectMatchedOptions() {
  matcher
    .MatchedGroupIds()
    .then(matchedIds => {
      matchedIds.forEach(gid => {
        const group = groupStore.groups.find(g => g.id === gid)
        if (!group) return
        if (group.type === 'network') {
          selectedNetwork.value = gid
        } else if (group.type === 'display') {
          selectedDisplay.value = gid
        } else if (group.type === 'miscellaneous') {
          if (!selectedMiscellaneous.value.includes(gid)) {
            selectedMiscellaneous.value.push(gid)
          }
        }
      })
    })
    .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
}

function resetSelection() {
  selectedNetwork.value = 0
  selectedDisplay.value = 0
  selectedMiscellaneous.value = []
}

async function handleSubmit() {
  const commands: Array<Command> = []

  if (settingStore.settings.set_password) {
    commands.push({
      id: 'set_password',
      groupName: t('taskSetPassword'),
      config: {
        program: 'powershell',
        options: [
          '-WindowStyle',
          'Hidden',
          '-Command',
          `Set-LocalUser -Name $Env:UserName -Password ${
            settingStore.settings.password == ''
              ? '(new-object System.Security.SecureString)'
              : `(ConvertTo-SecureString ${settingStore.settings.password} -AsPlainText -Force)`
          }`
        ],
        minExeTime: 0.5,
        allowRtCodes: [0],
        incompatibles: []
      }
    })
  }

  if (settingStore.settings.create_partition) {
    commands.push({
      id: 'create_partition',
      groupName: t('taskCreatePartitions'),
      config: {
        program: 'powershell',
        options: [
          '-WindowStyle',
          'Hidden',
          '-Command',
          'Get-Disk | Where-Object PartitionStyle -Eq "RAW" | Initialize-Disk -PassThru | New-Partition -AssignDriveLetter -UseMaximumSize | Format-Volume'
        ],
        minExeTime: 1,
        allowRtCodes: [0],
        incompatibles: []
      }
    })
  }

  groupStore.groups
    .filter(group =>
      [selectedNetwork.value, selectedDisplay.value, ...selectedMiscellaneous.value].includes(
        group.id
      )
    )
    .forEach(group => {
      group.drivers.forEach(driver => {
        commands.push({
          id: driver.id,
          name: driver.name,
          groupName: group.name,
          config: {
            program: driver.path,
            options: driver.flags,
            minExeTime: driver.minExeTime,
            allowRtCodes: driver.allowRtCodes,
            incompatibles: driver.incompatibles
          }
        })
      })
    })

  // Expand incompatibilities for mutually-exclusive driver groups
  commands.forEach(cmd => {
    // Find which group this driver belongs to
    const driverGroup = groupStore.groups.find(g => g.drivers.some(d => d.id === cmd.id))

    // If the group is marked as mutually exclusive, add all other drivers in that group as incompatible
    if (driverGroup?.mutuallyExclusive) {
      const otherDriverIds = driverGroup.drivers.filter(d => d.id !== cmd.id).map(d => d.id)

      // Add other drivers to incompatibles, avoiding duplicates
      otherDriverIds.forEach(id => {
        if (!cmd.config.incompatibles.includes(id)) {
          cmd.config.incompatibles.push(id)
        }
      })
    }
  })

  if (commands.length == 0) {
    toast.add({ title: t('warnNoInputWarning'), color: 'warning' })
    return
  }

  statusModal.value?.show(settingStore.settings.parallel_install, commands)
}

onBeforeUnmount(() => {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
})
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex flex-1 flex-col gap-y-1 overflow-y-auto rounded-sm border p-1">
      <template v-if="systemInfo.hw !== null && systemInfo.os !== null">
        <div v-for="[part, names] in Object.entries(systemInfo.hw)" :key="part">
          <h2 class="text-sm font-bold">
            {{ $t(hwKey(part)) }}
            <UBadge
              v-if="part === 'cpu' && (cpuTemp !== null || cpuTempError !== null)"
              size="sm"
              class="ms-1 rounded px-1.5 py-0 text-xs font-medium"
              :class="
                cpuTempError !== null
                  ? 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
                  : cpuTemp! < 45
                    ? 'bg-apple-green-100 text-apple-green-700'
                    : cpuTemp! >= 85
                      ? 'bg-red-100 text-red-700'
                      : 'bg-yellow-100 text-yellow-700'
              "
            >
              {{ cpuTempError !== null ? '--' : cpuTemp! }}°C
            </UBadge>
          </h2>

          <p v-for="(name, i) in names" :key="i" class="text-sm">
            {{ name }}
          </p>
        </div>

        <div>
          <h2 class="text-sm font-bold">{{ $t('hwOs') }}</h2>

          <p class="text-sm">
            {{ [systemInfo.os!.caption, systemInfo.os!.displayVersion].filter(Boolean).join(' ') }}
            —
            {{
              $t(systemInfo.os!.activated ? 'osActivationActivated' : 'osActivationNotActivated')
            }}
          </p>
        </div>
      </template>

      <template v-else>
        <div v-for="i in 7" :key="i">
          <h2 class="mb-1 h-5">
            <USkeleton class="h-4" :style="{ width: `${Math.random() * (25 - 15) + 15}%` }" />
          </h2>

          <p class="h-5">
            <USkeleton class="h-4" :style="{ width: `${Math.random() * (85 - 30) + 30}%` }" />
          </p>
        </div>
      </template>
    </div>

    <form ref="form" class="mt-3 flex h-28 gap-x-3">
      <div class="flex flex-1 flex-col justify-between">
        <div class="relative w-full">
          <label
            class="pointer-events-none absolute inset-s-4 top-0 h-full translate-y-1 text-xs text-gray-500"
          >
            {{ $t('categoryNetwork') }}
          </label>

          <select
            v-model="selectedNetwork"
            class="w-full rounded-lg border border-apple-green-300 bg-white px-3 pt-5 pb-1 text-sm focus:border-apple-green-600 focus:ring-1 focus:ring-apple-green-600 focus:outline-none"
          >
            <option :value="0">{{ $t('labelPleaseSelect') }}</option>

            <option v-for="g in groups.filter(g => g.type == 'network')" :key="g.id" :value="g.id">
              {{ g.name }}
            </option>
          </select>
        </div>

        <div class="relative w-full">
          <label
            class="pointer-events-none absolute inset-s-4 top-0 h-full translate-y-1 text-xs text-gray-500"
          >
            {{ $t('categoryDisplay') }}
          </label>

          <select
            v-model="selectedDisplay"
            class="w-full rounded-lg border border-apple-green-300 bg-white px-3 pt-5 pb-1 text-sm focus:border-apple-green-600 focus:ring-1 focus:ring-apple-green-600 focus:outline-none"
          >
            <option :value="0">{{ $t('labelPleaseSelect') }}</option>

            <option v-for="g in groups.filter(g => g.type == 'display')" :key="g.id" :value="g.id">
              {{ g.name }}
            </option>
          </select>
        </div>
      </div>

      <div class="flex flex-1">
        <div class="relative mb-3 h-full w-full">
          <label
            class="pointer-events-none absolute top-1 left-3 origin-top-left -translate-y-[0.55rem] scale-[0.9] bg-white px-2 text-xs text-gray-500"
          >
            {{ $t('categoryMiscellaneous') }}
          </label>

          <div class="h-full overflow-y-scroll rounded-lg border border-apple-green-300 px-2 pt-3">
            <template v-for="g in groups.filter(g => g.type == 'miscellaneous')" :key="g.id">
              <label class="flex w-full cursor-pointer items-center select-none">
                <UCheckbox
                  :model-value="selectedMiscellaneous.includes(g.id)"
                  color="primary"
                  class="me-1.5"
                  @update:model-value="
                    checked => {
                      if (checked) {
                        selectedMiscellaneous.push(g.id)
                      } else {
                        selectedMiscellaneous = selectedMiscellaneous.filter(id => id !== g.id)
                      }
                    }
                  "
                />
                {{ g.name }}
              </label>
            </template>
          </div>
        </div>
      </div>
    </form>

    <hr class="my-3" />

    <div class="flex gap-x-6">
      <div class="flex flex-col">
        <p class="font-semibold">{{ $t('titleTasksAndSettings') }}</p>

        <div class="flex flex-1 flex-col justify-around">
          <div class="flex gap-x-4">
            <label class="flex cursor-pointer items-center gap-x-1.5 select-none">
              <UCheckbox
                v-model="settingStore.settings.create_partition"
                name="create_partition"
                color="primary"
              />
              {{ $t('settingCreatePartitions') }}
            </label>

            <label class="flex cursor-pointer items-center gap-x-1.5 select-none">
              <UCheckbox
                v-model="settingStore.settings.parallel_install"
                name="parallel_install"
                color="primary"
              />
              {{ $t('settingParallelInstall') }}
            </label>
          </div>

          <div class="flex gap-x-2">
            <label class="flex cursor-pointer items-center gap-x-1.5 select-none">
              <UCheckbox
                v-model="settingStore.settings.set_password"
                name="set_password"
                color="primary"
              />
              {{ $t('settingSetPassword') }}
            </label>

            <UInput
              v-model="settingStore.settings.password"
              type="password"
              name="password"
              color="primary"
              size="sm"
              class="max-w-28"
              :disabled="!settingStore.settings.set_password"
            />
          </div>
        </div>
      </div>

      <div class="flex grow flex-col justify-between">
        <div>
          <label class="mb-1 block text-sm text-gray-900">
            {{ $t('settingSuccessAction') }}
          </label>

          <USelect
            v-model="settingStore.settings.success_action"
            name="success_action"
            class="w-full"
            color="primary"
            :items="[
              { label: $t('actionNothing'), value: 'nothing' },
              { label: $t('actionShutdown'), value: 'shutdown' },
              { label: $t('actionReboot'), value: 'reboot' },
              { label: $t('actionFirmware'), value: 'firmware' }
            ]"
          />
        </div>

        <div class="mt-2 flex h-8 flex-row items-center justify-end gap-x-3">
          <UButton type="button" color="neutral" variant="outline" @click="selectMatchedOptions">
            {{ $t('labelMatch') }}
          </UButton>

          <UButton type="button" color="secondary" variant="outline" @click="resetSelection">
            {{ $t('actionReset') }}
          </UButton>

          <UButton color="secondary" @click="handleSubmit">
            {{ $t('actionExecute') }}
          </UButton>
        </div>
      </div>
    </div>
  </div>

  <CommandStatueModal
    ref="statusModal"
    @completed="
      () => {
        loadSystemInfo()

        const delay = Math.max(
          0,
          Math.floor(Number(settingStore.settings.success_action_delay) || 0)
        )
        switch (settingStore.settings.success_action) {
          case 'shutdown':
            executor.RunAndOutput('cmd', ['/C', `shutdown /s /t ${delay}`], true)
            break
          case 'reboot':
            executor.RunAndOutput('cmd', ['/C', `shutdown /r /t ${delay}`], true)
            break
          case 'firmware':
            executor
              .RunAndOutput('cmd', ['/C', `shutdown /r /fw /t ${delay}`], true)
              .then(result => {
                if (result.exitCode != 0) {
                  executor.RunAndOutput('cmd', ['/C', `shutdown /r /fw /t ${delay}`], true)
                }
              })
            break
        }
      }
    "
  ></CommandStatueModal>
</template>
