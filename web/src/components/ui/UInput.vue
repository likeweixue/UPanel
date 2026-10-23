<template>
  <div v-if="revealable" class="input-affix">
    <input
      :id="id"
      v-bind="$attrs"
      :value="modelValue"
      :type="revealed ? 'text' : 'password'"
      class="input"
      :class="{ 'input--sm': size === 'sm' }"
      :aria-invalid="invalid ? 'true' : undefined"
      @input="$emit('update:modelValue', $event.target.value)"
    />
    <UButton
      class="input-affix__btn"
      variant="ghost"
      size="sm"
      icon-only
      :icon="revealed ? 'eye-off' : 'eye'"
      :aria-label="revealed ? '隐藏密码' : '显示密码'"
      @click="revealed = !revealed"
    />
  </div>
  <input
    v-else
    :id="id"
    v-bind="$attrs"
    :value="modelValue"
    :type="type"
    class="input"
    :class="{ 'input--sm': size === 'sm' }"
    :aria-invalid="invalid ? 'true' : undefined"
    @input="$emit('update:modelValue', $event.target.value)"
  />
</template>

<script setup>
import { ref } from 'vue'
import UButton from './UButton.vue'

defineOptions({ inheritAttrs: false })

defineProps({
  modelValue: { type: [String, Number], default: '' },
  id: { type: String, default: '' },
  type: { type: String, default: 'text' },
  size: { type: String, default: 'md' },
  invalid: { type: Boolean, default: false },
  revealable: { type: Boolean, default: false },
})

defineEmits(['update:modelValue'])

const revealed = ref(false)
</script>
