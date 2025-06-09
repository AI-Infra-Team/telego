<template>
  <div class="bg-white shadow-sm border-b border-gray-200">
    <div class="px-6 py-4">
      <div class="flex items-center justify-between">
        <!-- 面包屑导航 -->
        <div class="flex items-center space-x-2">
          <el-button
            v-if="!sidebarOpen"
            @click="toggleSidebar"
            link
            class="lg:hidden"
          >
            <el-icon><Menu /></el-icon>
          </el-button>
          
          <el-breadcrumb separator="/">
            <el-breadcrumb-item
              v-for="(item, index) in breadcrumbs"
              :key="index"
              :to="item.path && index < breadcrumbs.length - 1 ? { path: item.path } : null"
            >
              {{ item.label }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <!-- 右侧操作区 -->
        <div class="flex items-center space-x-2">
          <!-- 刷新按钮 -->
          <el-button
            @click="refreshPage"
            :loading="isRefreshing"
            link
            title="刷新页面"
          >
            <el-icon><Refresh /></el-icon>
          </el-button>

          <!-- 通知按钮 -->
          <el-badge :value="3" type="danger">
            <el-button link title="通知">
              <el-icon><Bell /></el-icon>
            </el-button>
          </el-badge>

          <!-- 全屏切换 -->
          <el-button
            @click="toggleFullscreen"
            link
            title="全屏切换"
          >
            <el-icon>
              <FullScreen v-if="!isFullscreen" />
              <Close v-else />
            </el-icon>
          </el-button>

          <!-- 用户菜单 -->
          <el-dropdown trigger="click">
            <el-button link class="flex items-center space-x-2">
              <el-avatar size="small" style="background-color: #3b82f6;">A</el-avatar>
              <el-icon><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>个人资料</el-dropdown-item>
                <el-dropdown-item>账户设置</el-dropdown-item>
                <el-dropdown-item divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  Menu,
  Refresh,
  Bell,
  FullScreen,
  Close,
  ArrowDown
} from '@element-plus/icons-vue'

// Props
interface Props {
  sidebarOpen?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  sidebarOpen: true
})

// Emits
const emit = defineEmits<{
  toggleSidebar: []
}>()

// State
const route = useRoute()
const isRefreshing = ref(false)
const isFullscreen = ref(false)

// 面包屑导航配置
const breadcrumbConfig: Record<string, { label: string; parent?: string }> = {
  '/': { label: '初始化' },
  '/dashboard': { label: '概览' },
  '/deployments': { label: '部署管理' },
  '/monitoring': { label: '监控' },
  '/logs': { label: '日志' },
  '/security': { label: '安全' },
  '/settings': { label: '设置' },
}

// 计算面包屑
const breadcrumbs = computed(() => {
  const currentPath = route.path
  const config = breadcrumbConfig[currentPath]
  
  if (!config) {
    return [{ label: '未知页面', path: currentPath }]
  }

  const crumbs = []
  
  // 添加首页
  if (currentPath !== '/') {
    crumbs.push({ label: 'Telego', path: '/' })
  }
  
  // 添加当前页面
  crumbs.push({ label: config.label, path: currentPath })
  
  return crumbs
})

// 方法
const toggleSidebar = () => {
  emit('toggleSidebar')
}

const refreshPage = async () => {
  isRefreshing.value = true
  try {
    // 这里可以添加页面刷新逻辑
    await new Promise(resolve => setTimeout(resolve, 1000))
    window.location.reload()
  } finally {
    isRefreshing.value = false
  }
}

const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}
</script>

<style scoped>
/* Element Plus样式覆盖 */
:deep(.el-breadcrumb__inner) {
  color: #6b7280;
  font-weight: 500;
}

:deep(.el-breadcrumb__inner.is-link:hover) {
  color: #374151;
}
</style> 