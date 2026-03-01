import { writable } from 'svelte/store'
import { runtimeModeConfig } from '../runtime/config/mode-config.js'

export const structure = writable({})
export const firstLoad = writable(true)
export const connected = writable(false)
export const URBIT_MODE = writable(runtimeModeConfig.urbitMode)
export const DEV_PANEL = writable(runtimeModeConfig.devPanel)
export const startramMaxReminderDays = writable(7)
