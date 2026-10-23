<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal" @keydown="onKeydown">
        <div class="modal__backdrop" @click="onBackdrop" />
        <div
          ref="dialogRef"
          class="modal__dialog"
          :class="sizeClass"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :aria-describedby="description ? descId : undefined"
          tabindex="-1"
        >
          <header class="modal__header">
            <div>
              <h2 :id="titleId" class="modal__title">{{ title }}</h2>
              <p v-if="description" :id="descId" class="modal__desc">{{ description }}</p>
            </div>
            <UButton
              class="modal__close"
              variant="ghost"
              icon-only
              icon="x"
              aria-label="关闭对话框"
              :disabled="busy"
              @click="close"
            />
          </header>
          <div class="modal__body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="modal__footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import UButton from './UButton.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  size: { type: String, default: 'md' },
  closeOnBackdrop: { type: Boolean, default: true },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'close'])

let seq = 0
const uid = `modal-${++seq}-${Math.random().toString(36).slice(2, 7)}`
const titleId = `${uid}-title`
const descId = `${uid}-desc`

const dialogRef = ref(null)
let lastActive = null
let previousOverflow = ''

const sizeClass = computed(() =>
  props.size === 'sm' ? 'modal__dialog--sm' : props.size === 'lg' ? 'modal__dialog--lg' : props.size === 'xl' ? 'modal__dialog--xl' : ''
)

function focusables() {
  if (!dialogRef.value) return []
  return Array.from(
    dialogRef.value.querySelectorAll(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  ).filter((el) => el.offsetParent !== null || el === document.activeElement)
}

function close() {
  if (props.busy) return
  emit('update:modelValue', false)
  emit('close')
}

function onBackdrop() {
  if (props.closeOnBackdrop && !props.busy) close()
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    close()
    return
  }
  if (event.key !== 'Tab') return
  const items = focusables()
  if (items.length === 0) {
    event.preventDefault()
    dialogRef.value?.focus()
    return
  }
  const first = items[0]
  const last = items[items.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      lastActive = document.activeElement
      previousOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      await nextTick()
      const items = focusables()
      const target = dialogRef.value?.querySelector('[autofocus]') || items[0] || dialogRef.value
      target?.focus?.()
    } else {
      document.body.style.overflow = previousOverflow || ''
      if (lastActive && typeof lastActive.focus === 'function') {
        setTimeout(() => lastActive.focus(), 0)
      }
    }
  }
)

onBeforeUnmount(() => {
  document.body.style.overflow = previousOverflow || ''
})
</script>
