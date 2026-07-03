<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ title?: string }>()

const tags = defineModel<Array<string>>({ default: [] })

const input = ref<string>('')

function pushTag() {
  const value = input.value.trim()
  if (value) {
    if (!tags.value.some(t => t.toLowerCase() === value.toLowerCase())) {
      tags.value.push(value)
    }
    input.value = ''
  }
}

function removeTag(index: number) {
  tags.value.splice(index, 1)
}
</script>

<template>
  <div class="flex flex-wrap gap-1 p-1">
    <UButton v-for="(tag, i) in tags" :key="tag" size="sm" type="button" @click="removeTag(i)">
      {{ tag }}
      <Icon icon="mdi:close" class="h-6 w-6" />
    </UButton>

    <UInput
      v-model="input"
      type="text"
      class="grow"
      size="sm"
      @keydown.backspace="
        () => {
          if (input.value.length == 0) {
            removeTag(tags.length - 1)
          }
        }
      "
      @keydown.enter="
        (event: KeyboardEvent) => {
          if (input.value != '') {
            event.preventDefault()
          }
          pushTag()
        }
      "
    />
  </div>
</template>
