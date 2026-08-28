<script setup lang="ts">
import * as executor from '@/wailsjs/go/execute/CommandExecutor'
import { decodeError } from '@/utils/index'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const toast = useToast()
const powerActionModal = ref(false)
const powerActionType = ref<'shutdown' | 'reboot' | 'firmware'>('shutdown')

function confirmPowerAction() {
  powerActionModal.value = false
  const delay = 0
  switch (powerActionType.value) {
    case 'shutdown':
      executor.RunAndOutput('cmd', ['/C', `shutdown /s /t ${delay}`], true)
      break
    case 'reboot':
      executor.RunAndOutput('cmd', ['/C', `shutdown /r /t ${delay}`], true)
      break
    case 'firmware':
      executor.RunAndOutput('cmd', ['/C', `shutdown /r /fw /t ${delay}`], true)
      break
  }
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-6 overflow-y-auto p-2">
    <PageHeader
      variant="normal"
      :title="$t('titleSetupTools')"
      :description="$t('descSetupTools')"
    />

    <div>
      <div class="mt-4">
        <h2 class="mb-2 font-semibold">{{ $t('toolComputerManagement') }}</h2>

        <div class="grid grid-cols-2 gap-2">
          <UButton
            leading-icon="mdi:desktop-classic"
            color="primary"
            variant="link"
            size="sm"
            @click="
              executor.RunAndOutput('mmc.exe', ['compmgmt.msc'], false).catch(error =>
                toast.add({
                  title: t('errToolOpenFailed'),
                  description: decodeError(error, t),
                  color: 'error',
                  icon: 'mdi:cross-circle-outline'
                })
              )
            "
          >
            {{ $t('toolComputerManagement') }}
          </UButton>

          <UButton
            leading-icon="mdi:monitor"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor.RunAndOutput('mmc.exe', ['devmgmt.msc'], false).catch(error =>
                toast.add({
                  title: t('errToolOpenFailed'),
                  description: decodeError(error, t),
                  color: 'error',
                  icon: 'mdi:cross-circle-outline'
                })
              )
            "
          >
            {{ $t('toolDeviceManager') }}
          </UButton>

          <UButton
            leading-icon="mdi:harddisk"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor.RunAndOutput('mmc.exe', ['diskmgmt.msc'], false).catch(error =>
                toast.add({
                  title: t('errToolOpenFailed'),
                  description: decodeError(error, t),
                  color: 'error',
                  icon: 'mdi:cross-circle-outline'
                })
              )
            "
          >
            {{ $t('toolDiskManager') }}
          </UButton>
        </div>
      </div>

      <div class="mt-6">
        <h2 class="mb-2 font-semibold">{{ $t('toolSettings') }}</h2>

        <div class="grid grid-cols-2 gap-2">
          <UButton
            leading-icon="mdi:settings-outline"
            color="primary"
            variant="link"
            size="sm"
            @click="
              executor.RunAndOutput('cmd', ['/c', 'start', 'ms-settings:'], false).catch(error =>
                toast.add({
                  title: t('errToolOpenFailed'),
                  description: decodeError(error, t),
                  color: 'error',
                  icon: 'mdi:cross-circle-outline'
                })
              )
            "
          >
            {{ $t('toolWindowsSettings') }}
          </UButton>

          <UButton
            leading-icon="mdi:key"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor
                .RunAndOutput('cmd', ['/c', 'start', 'ms-settings:activation'], false)
                .catch(error =>
                  toast.add({
                    title: t('errToolOpenFailed'),
                    description: decodeError(error, t),
                    color: 'error',
                    icon: 'mdi:cross-circle-outline'
                  })
                )
            "
          >
            {{ $t('toolWindowsActivation') }}
          </UButton>

          <UButton
            leading-icon="mdi:update"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor
                .RunAndOutput('cmd', ['/c', 'start', 'ms-settings:windowsupdate'], false)
                .catch(error =>
                  toast.add({
                    title: t('errToolOpenFailed'),
                    description: decodeError(error, t),
                    color: 'error',
                    icon: 'mdi:cross-circle-outline'
                  })
                )
            "
          >
            {{ $t('toolWindowsUpdate') }}
          </UButton>

          <UButton
            leading-icon="mdi:package-variant-closed"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor
                .RunAndOutput('cmd', ['/c', 'start', 'ms-settings:appsfeatures'], false)
                .catch(error =>
                  toast.add({
                    title: t('errToolOpenFailed'),
                    description: decodeError(error, t),
                    color: 'error',
                    icon: 'mdi:cross-circle-outline'
                  })
                )
            "
          >
            {{ $t('toolInstalledApps') }}
          </UButton>

          <UButton
            leading-icon="mdi:wifi"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor
                .RunAndOutput('cmd', ['/c', 'start', 'ms-settings:network-wifi'], false)
                .catch(error =>
                  toast.add({
                    title: t('errToolOpenFailed'),
                    description: decodeError(error, t),
                    color: 'error',
                    icon: 'mdi:cross-circle-outline'
                  })
                )
            "
          >
            {{ $t('toolWifi') }}
          </UButton>

          <UButton
            leading-icon="mdi:bluetooth"
            color="secondary"
            variant="link"
            size="sm"
            @click="
              executor
                .RunAndOutput('cmd', ['/c', 'start', 'ms-settings:bluetooth'], false)
                .catch(error =>
                  toast.add({
                    title: t('errToolOpenFailed'),
                    description: decodeError(error, t),
                    color: 'error',
                    icon: 'mdi:cross-circle-outline'
                  })
                )
            "
          >
            {{ $t('toolBluetooth') }}
          </UButton>
        </div>
      </div>
    </div>

    <!-- Power Actions Section -->
    <div>
      <h1 class="font-bold">{{ $t('titleShutdownOptions') }}</h1>

      <div class="mt-4 grid grid-cols-3 gap-2">
        <UButton
          leading-icon="mdi:power"
          color="error"
          variant="link"
          size="sm"
          @click="
            () => {
              powerActionType = 'shutdown'
              powerActionModal = true
            }
          "
        >
          {{ $t('actionShutdown') }}
        </UButton>

        <UButton
          leading-icon="mdi:restart"
          color="warning"
          variant="link"
          size="sm"
          @click="
            () => {
              powerActionType = 'reboot'
              powerActionModal = true
            }
          "
        >
          {{ $t('actionReboot') }}
        </UButton>

        <UButton
          leading-icon="mdi:restart-alert"
          color="warning"
          variant="link"
          size="sm"
          @click="
            () => {
              powerActionType = 'firmware'
              powerActionModal = true
            }
          "
        >
          {{ $t('actionFirmware') }}
        </UButton>
      </div>
    </div>

    <UModal v-model:open="powerActionModal">
      <template #body>
        <div class="p-4">
          <div class="flex items-start gap-x-3">
            <UIcon name="i-lucide-alert-triangle" class="mt-0.5 h-6 w-6 shrink-0 text-yellow-500" />

            <div class="flex-1">
              <h2 class="text-lg font-semibold">
                {{
                  t(`action${powerActionType.charAt(0).toUpperCase() + powerActionType.slice(1)}`)
                }}?
              </h2>

              <p class="mt-1 text-sm text-gray-600">
                {{ $t('msgPowerActionsConfirm') }}
              </p>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-x-3">
            <UButton color="neutral" variant="ghost" @click="powerActionModal = false">
              {{ $t('cancel') }}
            </UButton>

            <UButton color="warning" variant="soft" @click="confirmPowerAction">
              {{ $t('confirm') }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
