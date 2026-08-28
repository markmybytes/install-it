<script setup lang="ts">
import { storage } from '@/wailsjs/go/models'
import * as appSettingStorage from '@/wailsjs/go/storage/AppSettingStorage'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { decodeError } from '@/utils/index'

const { t, locale } = useI18n()
const toast = useToast()

function settingKey(key: string): string {
  return `setting${key.charAt(0).toUpperCase() + key.slice(1)}`
}

function actionKey(action: string): string {
  return `action${action.charAt(0).toUpperCase() + action.slice(1)}`
}

const tabs = ref({ general: true, defaults: false, display: false })

const settingStore = useAppSettingStore()
const {
  data: settings,
  modified,
  reset
} = useEditor({ source: () => settingStore.settings, warnOnUnsavedLeave: true })

function askLeave() {
  const formStore = useUnsavedFormStore()
  if (!modified.value) {
    return Promise.resolve(true)
  }

  formStore.show = true
  return new Promise<boolean>(resolve => {
    formStore.setAnswerHandler(resolve)
  })
}

function handleSubmit() {
  appSettingStorage
    .Update(settings.value)
    .then(newAppSettings => {
      useAppSettingStore().settings = newAppSettings
      return reset()
    })
    .then(() => {
      locale.value = settings.value.language
      toast.add({ title: t('msgSaved'), color: 'success' })
    })
    .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-4 bg-transparent p-2 text-gray-900 dark:text-gray-100">
    <PageHeader variant="normal" :title="$t('titleSettings')" />

    <nav class="flex gap-x-1.5">
      <UButton
        v-for="key in Object.keys(tabs)"
        :key="key"
        type="button"
        :color="tabs[key as keyof typeof tabs] ? 'secondary' : 'neutral'"
        :variant="tabs[key as keyof typeof tabs] ? 'solid' : 'ghost'"
        class="font-semibold"
        @click="
          askLeave().then(leave => {
            if (leave) {
              reset()
              Object.keys(tabs).forEach(k => (tabs[k as keyof typeof tabs] = k === key))
            }
          })
        "
      >
        {{ $t(settingKey(key)) }}
      </UButton>
    </nav>

    <div class="flex flex-1 flex-col overflow-y-auto px-4 py-4">
      <div class="max-w-3xl">
        <div v-show="tabs.general" class="flex flex-col gap-y-4">
          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('labelUpdateOptions') }}</h3>

            <div class="flex flex-col gap-y-3">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.auto_check_update"
                  name="auto_check_update"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingAutoCheckUpdate') }}</span>
              </label>

              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.allow_pre_release"
                  name="allow_pre_release"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingPreferPreRelease') }}</span>
              </label>
            </div>
          </section>

          <hr class="border-gray-100 dark:border-gray-800" />

          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingImportExport') }}</h3>

            <div class="flex flex-col gap-y-1">
              <label>{{ $t('settingImportUrl') }}</label>

              <UTextarea
                v-model="settings.driver_download_url"
                name="driver_download_url"
                color="primary"
                placeholder="https://"
                class="w-full max-w-lg"
                :rows="3"
              />
            </div>
          </section>
        </div>

        <div v-show="tabs.defaults" class="flex flex-col gap-y-4">
          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingTask') }}</h3>

            <div class="flex flex-col gap-y-3">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.create_partition"
                  name="create_partition"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingCreatePartitions') }}</span>
              </label>

              <div class="flex flex-col gap-y-2">
                <label class="flex w-fit cursor-pointer items-center select-none">
                  <UCheckbox
                    v-model="settings.set_password"
                    name="set_password"
                    color="primary"
                    class="me-2"
                  />

                  <span>{{ $t('settingSetPassword') }}</span>
                </label>

                <div
                  class="ml-6 transition-opacity duration-200"
                  :class="{ 'opacity-50': !settings.set_password }"
                >
                  <UInput
                    v-model="settings.password"
                    type="password"
                    name="password"
                    placeholder="••••••••"
                    color="primary"
                    class="w-56"
                    :disabled="!settings.set_password"
                  />
                </div>
              </div>
            </div>
          </section>

          <hr class="border-gray-100 dark:border-gray-800" />

          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingInstall') }}</h3>

            <div class="flex flex-col gap-y-4">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.parallel_install"
                  name="parallel_install"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingParallelInstall') }}</span>
              </label>

              <div class="flex flex-wrap items-start gap-4">
                <div class="flex flex-col gap-y-1">
                  <label>{{ $t('settingSuccessAction') }}</label>

                  <USelect
                    v-model="settings.success_action"
                    name="success_action"
                    color="primary"
                    class="w-56"
                    :items="
                      Object.values(storage.SuccessAction).map((action: string) => ({
                        label: $t(actionKey(action)),
                        value: action
                      }))
                    "
                  />
                </div>

                <div
                  class="flex flex-col gap-y-1 transition-opacity duration-200"
                  :class="{
                    'pointer-events-none opacity-40': settings.success_action === 'nothing'
                  }"
                >
                  <label>{{ $t('settingSuccessActionDelay') }}</label>

                  <div class="flex items-center gap-x-2">
                    <UInput
                      v-model="settings.success_action_delay"
                      type="number"
                      name="success_action_delay"
                      min="0"
                      color="primary"
                      class="w-24"
                      :disabled="settings.success_action === 'nothing'"
                      required
                    />

                    <span class="text-sm text-gray-500">{{ $t('labelSeconds') }}</span>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>

        <div v-show="tabs.display" class="flex flex-col gap-y-4">
          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingLanguage') }}</h3>

            <div class="flex flex-col gap-y-1">
              <USelect
                v-model="settings.language"
                name="language"
                color="primary"
                variant="outline"
                class="w-56"
                :items="[
                  { label: 'English', value: 'en' },
                  { label: '繁體中文', value: 'zh_Hant_HK' }
                ]"
              />
            </div>
          </section>

          <hr class="border-gray-100 dark:border-gray-800" />

          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingHardwareInfo') }}</h3>

            <div class="flex flex-col gap-y-3">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.filter_miniport_nic"
                  name="filter_miniport_nic"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingHideMiniportNic') }}</span>
              </label>

              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.filter_microsoft_nic"
                  name="filter_microsoft_nic"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingHideMicrosoftNic') }}</span>
              </label>
            </div>
          </section>

          <hr class="border-gray-100 dark:border-gray-800" />

          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingCPUTemp') }}</h3>

            <div class="flex flex-col gap-y-3">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.enable_cpu_temp"
                  name="enable_cpu_temp"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingEnableCPUTemp') }}</span>
              </label>

              <p class="text-xs text-gray-500">{{ $t('settingCPUTempRestartHint') }}</p>

              <div
                class="flex flex-col gap-y-1 transition-opacity duration-200"
                :class="{ 'pointer-events-none opacity-40': !settings.enable_cpu_temp }"
              >
                <label>{{ $t('settingCPUTempRefreshInterval') }}</label>

                <div class="flex items-center gap-x-2">
                  <USelect
                    v-model="settings.cpu_temp_refresh_interval"
                    name="cpu_temp_refresh_interval"
                    color="primary"
                    class="w-24"
                    :items="[1, 5, 10, 30, 60].map(v => ({ label: String(v), value: v }))"
                    :disabled="!settings.enable_cpu_temp"
                  />

                  <span class="text-sm text-gray-500">{{ $t('labelSeconds') }}</span>
                </div>
              </div>
            </div>
          </section>

          <hr class="border-gray-100 dark:border-gray-800" />

          <section>
            <h3 class="mb-3 text-lg font-bold">{{ $t('settingInstallOptions') }}</h3>

            <div class="flex flex-col gap-y-3">
              <label class="flex w-fit cursor-pointer items-center select-none">
                <UCheckbox
                  v-model="settings.hide_not_found"
                  name="hide_not_found"
                  color="primary"
                  class="me-2"
                />

                <span>{{ $t('settingHideNotFound') }}</span>
              </label>
            </div>
          </section>
        </div>
      </div>
    </div>

    <div class="flex shrink-0 items-center justify-end">
      <UButton type="button" color="secondary" class="px-6 font-medium" @click="handleSubmit">
        {{ $t('save') }}
      </UButton>
    </div>
  </div>
</template>
