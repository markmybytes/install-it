import { ExecutableExists } from '@/wailsjs/go/main/App'
import { storage } from '@/wailsjs/go/models'
import { defineStore } from 'pinia'

export default defineStore('driverGroup', () => {
  const groups = ref<storage.DriverGroup[]>([])
  const notFoundDrivers = ref<Array<number>>([])

  const findNotExists = (drivers: Array<storage.Driver>) =>
    Promise.all(
      drivers.map(d => ExecutableExists(d.path).then(exist => ({ id: d.id, exist: exist })))
    ).then(results => {
      return results
        .map(result => (result.exist ? undefined : result.id))
        .filter(v => v !== undefined)
    })

  watch(
    groups,
    newGroups =>
      findNotExists(newGroups.flatMap(g => g.drivers)).then(ids => (notFoundDrivers.value = ids)),
    { immediate: true }
  )

  return {
    groups,
    notFoundDrivers,
    isAllDriversExist: (g: storage.DriverGroup) =>
      g.drivers.map(d => d.id).every(id => !notFoundDrivers.value.includes(id))
  }
})
