<template>
  <div>
    <!-- 当前步骤 -->
    <el-card 
      :class="[
        'step-card transition-all duration-200',
        { 'ml-6': isChildStep }
      ]"
      :style="{ marginLeft: `${depth * 24}px` }"
    >
      <div class="flex items-start justify-between">
        <div class="flex-1">
          <div class="flex items-center mb-3">
            <!-- 展开/折叠按钮（如果有子步骤） -->
            <el-button
              v-if="hasChildren"
              @click="toggleExpanded"
              :icon="expanded ? ArrowDown : ArrowRight"
              size="small"
              text
              class="mr-2 p-1"
            />
            <div v-else class="w-8"></div>
            
            <!-- 步骤图标 -->
            <div class="flex items-center mr-3">
              <el-icon 
                :size="depth === 0 ? 24 : 20"
                :style="{ color: getStepIconColor(step.status) }"
              >
                <SuccessFilled v-if="step.status === 'completed'" />
                <Warning v-else-if="step.status === 'error'" />
                <Loading v-else-if="step.status === 'running'" class="rotating" />
                <Clock v-else />
              </el-icon>
            </div>
            
            <!-- 步骤信息 -->
            <div>
              <h3 :class="[
                'font-medium text-gray-900',
                depth === 0 ? 'text-lg' : 'text-base'
              ]">
                {{ step.name }}
              </h3>
              <p class="text-gray-600 text-sm">{{ step.description }}</p>
            </div>
          </div>

          <!-- 步骤进度条 -->
          <div class="mb-3 pl-10">
            <el-progress 
              :percentage="step.progress"
              :status="getStepProgressStatus(step.status)"
              :stroke-width="6"
              :show-text="false"
            />
            <div class="text-xs text-gray-500 mt-1">{{ step.progress }}%</div>
          </div>

          <!-- 错误信息 -->
          <el-alert
            v-if="step.error"
            :title="step.error"
            type="error"
            :closable="false"
            class="mt-2 ml-10"
            size="small"
          />
        </div>

        <!-- 重试按钮 -->
        <div v-if="step.status === 'error'" class="ml-4">
          <el-button 
            @click="$emit('retryStep', step.id)"
            :loading="loading"
            type="primary"
            size="small"
            :icon="Refresh"
          >
            重试
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 子步骤（递归渲染） -->
    <div v-if="hasChildren && expanded" class="mt-2">
      <StepTreeNode
        v-for="childStep in step.children"
        :key="childStep.id"
        :step="childStep"
        :depth="depth + 1"
        :loading="loading"
        @retry-step="$emit('retryStep', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { 
  SuccessFilled,
  Warning,
  Clock,
  Loading,
  Refresh,
  ArrowDown,
  ArrowRight
} from '@element-plus/icons-vue'
import type { InitializationStep } from '@/types'

interface Props {
  step: InitializationStep
  depth?: number  // 深度层级，用于样式控制
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  depth: 0
})

defineEmits<{
  retryStep: [stepId: string]
}>()

const expanded = ref(true) // 默认展开

// 是否有子步骤
const hasChildren = computed(() => {
  return props.step.children && props.step.children.length > 0
})

// 是否为子步骤
const isChildStep = computed(() => {
  return props.depth > 0
})

// 切换展开状态
function toggleExpanded() {
  expanded.value = !expanded.value
}

function getStepIconColor(stepStatus: string): string {
  switch (stepStatus) {
    case 'completed': return '#10b981'
    case 'running': return '#3b82f6'
    case 'error': return '#ef4444'
    default: return '#6b7280'
  }
}

function getStepProgressStatus(stepStatus: string) {
  switch (stepStatus) {
    case 'completed': return 'success'
    case 'error': return 'exception'
    default: return undefined
  }
}
</script>

<style scoped>
.step-card {
  transition: all 0.2s ease;
}

.step-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.step-card {
  border-left: 4px solid #3b82f6;
}

.ml-6 .step-card {
  border-left: 3px solid #10b981;
  background-color: #f8fafc;
}

.rotating {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style> 