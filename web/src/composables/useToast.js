import { reactive } from 'vue'

export const toastState = reactive({ items: [] })
let seq = 0

const ICONS = {
  success: 'check',
  danger: 'alert',
  warning: 'alert',
  info: 'info',
}

export function removeToast(id) {
  const index = toastState.items.findIndex((item) => item.id === id)
  if (index >= 0) toastState.items.splice(index, 1)
}

export function pushToast(options) {
  const item = {
    id: ++seq,
    type: options.type || 'info',
    title: options.title || '',
    description: options.description || '',
    icon: ICONS[options.type || 'info'] || 'info',
    duration: options.duration == null ? 4200 : options.duration,
  }
  toastState.items.push(item)
  if (toastState.items.length > 4) toastState.items.shift()
  if (item.duration > 0) {
    setTimeout(() => removeToast(item.id), item.duration)
  }
  return item.id
}

export function toastSuccess(title, description) {
  return pushToast({ type: 'success', title, description })
}

export function toastError(title, description) {
  return pushToast({ type: 'danger', title, description, duration: 7000 })
}

export function toastWarning(title, description) {
  return pushToast({ type: 'warning', title, description })
}

export function toastInfo(title, description) {
  return pushToast({ type: 'info', title, description })
}

export function useToast() {
  return { toastState, pushToast, removeToast, toastSuccess, toastError, toastWarning, toastInfo }
}
