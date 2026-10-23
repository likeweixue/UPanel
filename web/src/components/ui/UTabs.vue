<template>
  <div>
    <div class="tabs" role="tablist" :aria-label="ariaLabel" @keydown="onKeydown">
      <button
        v-for="tab in tabs"
        :id="tabId(tab.id)"
        :key="tab.id"
        type="button"
        role="tab"
        class="tabs__tab"
        :aria-selected="modelValue === tab.id ? 'true' : 'false'"
        :aria-controls="panelId"
        :tabindex="modelValue === tab.id ? 0 : -1"
        :disabled="tab.disabled"
        @click="select(tab)"
      >
        <UIcon v-if="tab.icon" :name="tab.icon" :size="15" />
        <span>{{ tab.label }}</span>
        <span v-if="tab.count != null" class="tabs__count">{{ tab.count }}</span>
      </button>
    </div>
    <div
      :id="panelId"
      class="tabs__panel"
      role="tabpanel"
      :aria-labelledby="tabId(modelValue)"
      tabindex="0"
    >
      <slot />
    </div>
  </div>
</template>

<script setup>
import UIcon from './UIcon.vue'

const props = defineProps({
  tabs: { type: Array, required: true },
  modelValue: { type: String, required: true },
  ariaLabel: { type: String, default: '标签页' },
})

const emit = defineEmits(['update:modelValue', 'change'])
const uid = `tabs-${Math.random().toString(36).slice(2, 9)}`

const tabId = (id) => `${uid}-tab-${id}`
const panelId = `${uid}-panel`

function select(tab) {
  if (tab.disabled || tab.id === props.modelValue) return
  emit('update:modelValue', tab.id)
  emit('change', tab.id)
}

function onKeydown(event) {
  const enabled = props.tabs.filter((tab) => !tab.disabled)
  const index = enabled.findIndex((tab) => tab.id === props.modelValue)
  if (index < 0) return
  let next = index
  if (event.key === 'ArrowRight') next = (index + 1) % enabled.length
  else if (event.key === 'ArrowLeft') next = (index - 1 + enabled.length) % enabled.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = enabled.length - 1
  else return
  event.preventDefault()
  const target = enabled[next]
  emit('update:modelValue', target.id)
  emit('change', target.id)
  requestAnimationFrame(() => document.getElementById(tabId(target.id))?.focus())
}
</script>
