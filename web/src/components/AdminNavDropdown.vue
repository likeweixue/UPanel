<template>
  <div ref="rootRef" class="dropdown nav-dropdown">
    <button
      ref="triggerRef"
      type="button"
      class="dropdown__trigger"
      aria-haspopup="dialog"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="panelId"
      aria-label="打开后台导航，搜索全部功能"
      @click="toggle"
      @keydown.down.prevent="openPanel"
    >
      <span class="app-header__nav-label">
        <UIcon name="grid" :size="16" />
        <span class="app-header__nav-text app-header__nav-prefix">后台导航</span>
        <span v-if="currentLabel" class="app-header__nav-text app-header__nav-current">
          <span class="app-header__nav-sep">· </span>{{ currentLabel }}
        </span>
      </span>
      <UIcon name="chevron-down" :size="15" class="dropdown__chevron" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        :id="panelId"
        ref="panelRef"
        class="dropdown__panel"
        :style="panelStyle"
        role="dialog"
        aria-modal="false"
        aria-label="后台导航"
        @mousedown.stop
        @keydown.esc.stop.prevent="closePanel"
      >
        <div class="dropdown__search">
          <UIcon name="search" :size="16" class="muted" />
          <input
            ref="searchRef"
            v-model="query"
            class="input"
            type="text"
            role="combobox"
            aria-expanded="true"
            :aria-controls="listId"
            :aria-activedescendant="activeId"
            aria-autocomplete="list"
            placeholder="搜索功能、页面或关键词，例如 SSL、Redis、防火墙…"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="activate"
            @keydown.home.prevent="setActive(0)"
            @keydown.end.prevent="setActive(flatOptions.length - 1)"
            @keydown.tab="closePanel"
          />
          <span class="badge">{{ navCount }}</span>
        </div>

        <div class="nav-dropdown__list" role="listbox" :id="listId" aria-label="全部后台功能">
          <template v-for="group in filteredGroups" :key="group.id">
            <div class="nav-dropdown__group" role="presentation">{{ group.label }}</div>
            <button
              v-for="item in group.items"
              :id="optionId(item.id)"
              :key="item.id"
              type="button"
              role="option"
              class="nav-dropdown__option"
              :class="{ 'is-active': activeKey === item.id }"
              :aria-selected="activeKey === item.id ? 'true' : 'false'"
              @click="navigate(item)"
              @mouseenter="setActiveByKey(item.id)"
            >
              <UIcon :name="item.icon" :size="16" class="nav-dropdown__icon" />
              <span class="grow">
                <span class="nav-dropdown__label">{{ item.label }}</span>
                <span class="nav-dropdown__desc">{{ item.description }}</span>
              </span>
              <span v-if="isCurrent(item)" class="badge badge--primary">当前</span>
            </button>
          </template>
          <p v-if="flatOptions.length === 0" class="dropdown__empty">
            没有匹配的功能，试试其他关键词
          </p>
        </div>

        <div class="dropdown__footer">
          共 {{ navCount }} 个入口 · ↑↓ 选择 · Enter 打开 · Esc 关闭
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UIcon from '@/components/ui/UIcon.vue'
import { navCount, navGroups } from '@/nav/catalog'

const props = defineProps({
  currentLabel: { type: String, default: '' },
})

const emit = defineEmits(['navigate'])

const router = useRouter()
const route = useRoute()

const uid = `nav-${Math.random().toString(36).slice(2, 9)}`
const panelId = `${uid}-panel`
const listId = `${uid}-list`
const optionId = (id) => `${uid}-item-${id}`

const rootRef = ref(null)
const triggerRef = ref(null)
const panelRef = ref(null)
const searchRef = ref(null)
const open = ref(false)
const query = ref('')
const activeKey = ref('')
const panelStyle = ref({})

const filteredGroups = computed(() => {
  const q = query.value.trim().toLowerCase()
  return navGroups
    .map((group) => {
      const items = group.items.filter((item) => {
        if (!q) return true
        const haystack = [item.label, item.description, group.label, item.keywords.join(' ')]
          .join(' ')
          .toLowerCase()
        return q.split(/\s+/).every((word) => haystack.includes(word))
      })
      return { ...group, items }
    })
    .filter((group) => group.items.length > 0)
})

const flatOptions = computed(() => filteredGroups.value.flatMap((group) => group.items))

watch(query, () => {
  const first = flatOptions.value[0]
  activeKey.value = first ? first.id : ''
  scrollActiveIntoView()
})

const activeId = computed(() => {
  const item = flatOptions.value.find((entry) => entry.id === activeKey.value)
  return item ? optionId(item.id) : undefined
})

function isCurrent(item) {
  if (item.to !== route.path) return false
  const q = item.query || {}
  return Object.keys(q).every((key) => String(route.query[key] || '') === String(q[key]))
}

async function openPanel() {
  if (open.value) return
  open.value = true
  query.value = ''
  const current = navGroups.flatMap((group) => group.items).find(isCurrent)
  activeKey.value = current ? current.id : navGroups[0].items[0].id
  document.addEventListener('mousedown', onDocumentMouseDown, true)
  window.addEventListener('resize', positionPanel)
  window.addEventListener('scroll', positionPanel, true)
  await nextTick()
  positionPanel()
  searchRef.value?.focus()
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

function toggle() {
  if (open.value) closePanel()
  else openPanel()
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
  const width = Math.min(620, Math.max(280, viewportWidth - 24))
  const left = Math.max(8, Math.min(rect.left, viewportWidth - width - 8))
  const style = { left: `${left}px`, width: `${width}px`, top: `${rect.bottom + 6}px` }
  const maxHeight = window.innerHeight - rect.bottom - 24
  if (maxHeight < 260 && rect.top > maxHeight) {
    style.bottom = `${window.innerHeight - rect.top + 6}px`
    delete style.top
    style.maxHeight = `${Math.min(560, rect.top - 16)}px`
  } else {
    style.maxHeight = `${Math.min(560, Math.max(240, maxHeight))}px`
  }
  panelStyle.value = style
}

function setActive(index) {
  if (flatOptions.value.length === 0) return
  const next = Math.max(0, Math.min(index, flatOptions.value.length - 1))
  activeKey.value = flatOptions.value[next].id
  scrollActiveIntoView()
}

function setActiveByKey(key) {
  activeKey.value = key
}

function currentIndex() {
  return flatOptions.value.findIndex((item) => item.id === activeKey.value)
}

function move(delta) {
  if (flatOptions.value.length === 0) return
  const index = currentIndex()
  const next = index < 0 ? 0 : (index + delta + flatOptions.value.length) % flatOptions.value.length
  setActive(next)
}

function scrollActiveIntoView() {
  nextTick(() => {
    const id = activeId.value
    if (!id) return
    const el = document.getElementById(id)
    el?.scrollIntoView({ block: 'nearest' })
  })
}

function activate() {
  const item = flatOptions.value.find((entry) => entry.id === activeKey.value)
  if (item) navigate(item)
}

function navigate(item) {
  closePanel()
  emit('navigate', item)
  router.push({ path: item.to, query: item.query || {} })
}

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  window.removeEventListener('resize', positionPanel)
  window.removeEventListener('scroll', positionPanel, true)
  if (hadFocus) triggerRef.value?.focus()
})

defineExpose({ open: openPanel, close: closePanel })
</script>
