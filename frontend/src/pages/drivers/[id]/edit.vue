<script setup lang="ts">
import { useRoute } from 'vue-router'

const $route = useRoute()

const id = parseInt(($route.params as { id: string }).id)
</script>

<template>
  <DriverFormComponent :id="id" />
</template>
