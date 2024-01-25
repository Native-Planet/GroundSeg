import { writable } from 'svelte/store'
export const structure = writable({})
export const firstLoad = writable(true)
export const connected = writable(false)
export const URBIT_MODE = writable(true)  // set to true when running with vite.config.urbit.js
export const DEV_PANEL = writable(false)  // set to true when running with vite.config.urbit.js
export const startramMaxReminderDays = writable(7) // how many days before satellite icon shows triangle warning

export const daysUntilDate = (dateString) => {
  const targetDate = new Date(dateString);
  const currentDate = new Date();
  const diffTime = Math.abs(targetDate - currentDate);
  const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
};
