<template>
  <div>
    <div class="max-w-4xl mx-auto">
      <!-- 页面标题 -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-gray-900 mb-2">Telego 初始化状态监控</h1>
        <p class="text-gray-600">监控 Telego 系统的初始化流程状态</p>
      </div>

      <!-- 整体进度条 -->
      <el-card class="mb-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-semibold text-gray-900">整体进度</h2>
          <el-tag :type="overallStatusType" size="large">
            {{ overallStatusText }}
          </el-tag>
        </div>
        
        <el-progress 
          :percentage="status?.overallProgress || 0"
          :status="progressStatus"
          :stroke-width="12"
          class="mb-4"
        />
        
        <div class="text-sm text-gray-600">
          {{ status?.overallProgress || 0 }}% 完成
        </div>
      </el-card>

      <!-- 操作按钮 -->
      <div class="mb-6 flex gap-4">
        <el-button 
          @click="refreshStatus"
          :loading="loading"
          type="default"
          :icon="Refresh"
        >
          刷新状态
        </el-button>
        
        <el-button 
          @click="startInitialization"
          :loading="loading"
          :disabled="status?.overallStatus === 'running'"
          type="primary"
          :icon="VideoPlay"
        >
          开始初始化
        </el-button>
      </div>

      <!-- 步骤列表 - 树形结构 -->
      <div class="space-y-2">
        <template v-for="step in rootSteps" :key="step.id">
          <StepTreeNode 
            :step="step" 
            @retry-step="retryStep"
            :loading="loading"
          />
        </template>
      </div>

      <!-- 无数据状态 -->
      <el-empty 
        v-if="!status && !loading" 
        description="无法获取初始化状态"
      >
        <el-button type="primary" @click="refreshStatus">
          重新加载
        </el-button>
      </el-empty>

      <!-- 加载状态 -->
      <div v-if="loading && !status" class="text-center py-12">
        <el-icon size="32" class="rotating mb-4">
          <Loading />
        </el-icon>
        <p class="text-gray-500">加载中...</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { 
  SuccessFilled,
  Warning,
  Clock,
  Loading,
  Refresh,
  VideoPlay
} from '@element-plus/icons-vue'
import { initializationApi } from '@/api'
import type { InitializationStatus } from '@/types'
import StepTreeNode from '@/components/StepTreeNode.vue'

const status = ref<InitializationStatus | null>(null)
const loading = ref(false)

// 获取根级步骤（直接使用steps数组，因为现在是树形结构）
const rootSteps = computed(() => {
  if (!status.value?.steps) return []
  return status.value.steps
})

const overallStatusText = computed(() => {
  switch (status.value?.overallStatus) {
    case 'completed': return '已完成'
    case 'running': return '运行中'
    case 'error': return '错误'
    default: return '待开始'
  }
})

const overallStatusType = computed(() => {
  switch (status.value?.overallStatus) {
    case 'completed': return 'success'
    case 'running': return 'warning'
    case 'error': return 'danger'
    default: return 'info'
  }
})

const progressStatus = computed(() => {
  switch (status.value?.overallStatus) {
    case 'completed': return 'success'
    case 'error': return 'exception'
    default: return undefined
  }
})

async function refreshStatus() {
  loading.value = true
  try {
    const response = await initializationApi.getStatus()
    if (response.success && response.data) {
      status.value = response.data
    }
  } catch (error) {
    console.error('Failed to fetch status:', error)
  } finally {
    loading.value = false
  }
}

async function startInitialization() {
  loading.value = true
  try {
    const response = await initializationApi.startInitialization()
    if (response.success) {
      // 开始初始化后，立即刷新状态
      await refreshStatus()
      // 定时刷新状态直到完成
      const interval = setInterval(async () => {
        await refreshStatus()
        if (status.value?.overallStatus === 'completed' || status.value?.overallStatus === 'error') {
          clearInterval(interval)
        }
      }, 2000)
    }
  } catch (error) {
    console.error('Failed to start initialization:', error)
  } finally {
    loading.value = false
  }
}

async function retryStep(stepId: string) {
  loading.value = true
  try {
    const response = await initializationApi.retryStep(stepId)
    if (response.success) {
      // 重试后刷新状态
      await refreshStatus()
    }
  } catch (error) {
    console.error('Failed to retry step:', error)
  } finally {
    loading.value = false
  }
}

// 页面加载时获取初始状态
onMounted(() => {
  refreshStatus()
  
  // 设置定时刷新
  const interval = setInterval(refreshStatus, 5000)
  
  // 组件卸载时清除定时器
  onBeforeUnmount(() => {
    clearInterval(interval)
  })
})
</script>

<style scoped>
.step-card {
  transition: shadow 0.3s ease;
}

.step-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.rotating {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style> 