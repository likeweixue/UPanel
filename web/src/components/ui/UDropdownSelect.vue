<template>
  <div
    ref="rootRef"
    class="dropdown dropdown--select"
    :class="{ 'dropdown--block': block }"
  >
    <button
      ref="triggerRef"
      type="button"
      class="dropdown__trigger"
      :disabled="disabled"
      aria-haspopup="listbox"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="panelId"
      :aria-label="ariaLabel || undefined"
      @click="toggle"
      @keydown.down.prevent="openPanel()"
      @keydown.up.prevent="openPanel()"
      @keydown.esc="closePanel"
    >
      <span :class="['dropdown__value', { dropdown__placeholder: !selected }]">
        {{ selected ? selected.label : placeholder }}
      </span>
      <UIcon name="chevron-down" :size="15" class="dropdown__chevron" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        ref="panelRef"
        class="dropdown__panel"
        :style="panelStyle"
        @mousedown.stop
        @keydown.esc.stop="closePanel"
      >
        <div v-if="searchable" class="dropdown__search">
          <UIcon name="search" :size="15" class="muted" />
          <input
            ref="searchRef"
            v-model="query"
            class="input"
            type="text"
            role="combobox"
            :aria-expanded="'true'"
            :aria-controls="listId"
            :aria-activedescendant="activeId"
            aria-autocomplete="list"
            :placeholder="searchPlaceholder"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="selectActive"
            @keydown.home.prevent="setActive(0)"
            @keydown.end.prevent="setActive(optionRows.length - 1)"
          />
        </div>
        <ul
          :id="listId"
          ref="listRef"
          class="dropdown__list"
          role="listbox"
          :aria-label="ariaLabel || placeholder"
          :tabindex="searchable ? -1 : 0"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.enter.prevent="selectActive"
        >
          <template v-for="(row, index) in rows" :key="row.key">
            <li v-if="row.type === 'group'" class="dropdown__group-label" role="presentation">
              {{ row.label }}
            </li>
            <li v-else role="presentation">
              <button
                :id="optionId(row.optionIndex)"
                type="button"
                role="option"
                class="dropdown__option"
                :class="{ 'is-active': row.optionIndex === activeIndex }"
                :aria-selected="row.option.value === modelValue ? 'true' : 'false'"
                :disabled="row.option.disabled"
                @click="select(row.option)"
                @mouseenter="activeIndex = row.optionIndex"
              >
                <UIcon
                  v-if="row.option.icon"
                  :name="row.option.icon"
                  :size="15"
                  class="dropdown__option-icon"
                />
                <span class="dropdown__option-text">
                  <span>{{ row.option.label }}</span>
                  <span v-if="row.option.description" class="dropdown__option-desc">
                    {{ row.option.description }}
                  </span>
                </span>
                <UIcon
                  v-if="row.option.value === modelValue"
                  name="check"
                  :size="15"
                  class="dropdown__option-check"
                />
              </button>
            </li>
          </template>
          <li v-if="optionRows.length === 0" class="dropdown__empty">{{ emptyText }}</li>
        </ul>
        <div v-if="footer" class="dropdown__footer">{{ footer }}</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import UIcon from './UIcon.vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择' },
  searchPlaceholder: { type: String, default: '搜索…' },
  emptyText: { type: String, default: '没有匹配的选项' },
  searchable: { type: Boolean, default: true },
  block: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  ariaLabel: { type: String, default: '' },
  footer: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'change'])

const uid = `dd-${Math.random().toString(36).slice(2, 9)}`
const listId = `${uid}-list`
const panelId = `${uid}-panel`
const optionId = (index) => `${uid}-option-${index}`

const rootRef = ref(null)
const triggerRef = ref(null)
const panelRef = ref(null)
const searchRef = ref(null)
const listRef = ref(null)
const open = ref(false)
const query = ref('')
const activeIndex = ref(0)
const panelStyle = ref({})

const selected = computed(() => props.options.find((option) => option.value === props.modelValue))

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return props.options.filter((option) => {
    if (option.disabled && option.value !== props.modelValue) return false
    if (!q) return true
    return [option.label, option.description, option.group]
      .filter(Boolean)
      .some((text) => String(text).toLowerCase().includes(q))
  })
})

const rows = computed(() => {
  const out = []
  let lastGroup = null
  let optionIndex = 0
  for (const option of filtered.value) {
    if (option.group && option.group !== lastGroup) {
      out.push({ type: 'group', label: option.group, key: `group-${option.group}-${optionIndex}` })
      lastGroup = option.group
    }
    out.push({ type: 'option', option, optionIndex, key: `option-${option.value}-${optionIndex}` })
    optionIndex++
  }
  return out
})

const optionRows = computed(() => filtered.value)
const activeId = computed(() =>
  optionRows.value.length > 0 ? optionId(Math.min(activeIndex.value, optionRows.value.length - 1)) : undefined
)

watch(query, () => {
  activeIndex.value = 0
})

watch(
  () => props.modelValue,
  () => {
    if (open.value) {
      const index = optionRows.value.findIndex((option) => option.value === props.modelValue)
      if (index >= 0) activeIndex.value = index
    }
  }
)

function toggle() {
  if (props.disabled) return
  if (open.value) closePanel()
  else openPanel()
}

async function openPanel() {
  if (props.disabled || open.value) return
  query.value = ''
  const index = optionRows.value.findIndex((option) => option.value === props.modelValue)
  activeIndex.value = index >= 0 ? index : 0
  open.value = true
  document.addEventListener('mousedown', onDocumentMouseDown, true)
  window.addEventListener('resize', positionPanel)
  window.addEventListener('scroll', positionPanel, true)
  await nextTick()
  positionPanel()
  if (props.searchable) searchRef.value?.focus()
  else listRef.value?.focus()
  scrollActiveIntoView()
}

function closePanel() {
  if (!open.value) return
  const hadFocus = panelRef.value && panelRef.value.contains(document.activeElement)
  open.value = false
  query.value = ''
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
  if (hadFocus) triggerRef.value?.focus()
}

function onDocumentMouseDown(event) {
  const inTrigger = rootRef.value && rootRef.value.contains(event.target)
  const inPanel = panelRef.value && panelRef.value.contains(event.target)
  if (!inTrigger && !inPanel) closePanel()
}

function positionPanel() {
  if (!triggerRef.value) return
  const rect = triggerRef.value.getBoundingClientRect()
  const viewportWidth = document.documentElement.clientWidth
  const width = Math.min(viewportWidth - 16, Math.max(rect.width, 220))
  const left = Math.max(8, Math.min(rect.left, viewportWidth - width - 8))
  const spaceBelow = window.innerHeight - rect.bottom - 12
  const style = { left: `${left}px`, width: `${width}px` }
  if (spaceBelow < 220 && rect.top > spaceBelow) {
    style.bottom = `${window.innerHeight - rect.top + 6}px`
    style.maxHeight = `${Math.max(160, Math.min(340, rect.top - 16))}px`
  } else {
    style.top = `${rect.bottom + 6}px`
    style.maxHeight = `${Math.max(160, Math.min(340, spaceBelow))}px`
  }
  panelStyle.value = style
}

function move(delta) {
  if (optionRows.value.length === 0) return
  const next = (activeIndex.value + delta + optionRows.value.length) % optionRows.value.length
  activeIndex.value = next
  scrollActiveIntoView()
}

function setActive(index) {
  if (optionRows.value.length === 0) return
  activeIndex.value = Math.max(0, Math.min(index, optionRows.value.length - 1))
  scrollActiveIntoView()
}

function scrollActiveIntoView() {
  nextTick(() => {
    const el = document.getElementById(activeId.value || '')
    el?.scrollIntoView({ block: 'nearest' })
  })
}

function selectActive() {
  const option = optionRows.value[activeIndex.value]
  if (option) select(option)
}

function select(option) {
  if (option.disabled) return
  emit('update:modelValue', option.value)
  emit('change', option.value)
  closePanel()
  triggerRef.value?.focus()
}

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
})
</script>

<style scoped>
.dropdown__option-text {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
}

.dropdown__option-desc {
  color: var(--color-text-subtle);
  font-size: var(--text-sm);
  line-height: 1.4;
}

.dropdown__option-check {
  margin-left: auto;
  color: var(--color-primary);
}
</style>
