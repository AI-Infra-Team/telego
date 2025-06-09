<template>
  <div class="h-full flex flex-col bg-white shadow-lg">
    <!-- Logo 区域 -->
    <div class="flex items-center px-6 py-4 border-b border-gray-200">
      <div class="flex items-center">
        <el-avatar size="small" style="background-color: #3b82f6; color: white;">T</el-avatar>
        <div class="ml-3">
          <h1 class="text-lg font-semibold text-gray-900">Telego</h1>
          <p class="text-xs text-gray-500">管理控制台</p>
        </div>
      </div>
    </div>

    <!-- 导航菜单 -->
    <div class="flex-1 px-4 py-4">
      <el-menu
        :default-active="currentPath"
        class="border-none"
        router
      >
        <el-menu-item
          v-for="item in menuItems"
          :key="item.name"
          :index="item.path"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
          <el-badge
            v-if="item.badge"
            :value="item.badge"
            type="danger"
            class="ml-auto"
          />
        </el-menu-item>
      </el-menu>
    </div>

    <!-- 底部信息 -->
    <div class="px-4 py-4 border-t border-gray-200">
      <div class="flex items-center">
        <el-avatar size="small" style="background-color: #d1d5db;">
          <el-icon><User /></el-icon>
        </el-avatar>
        <div class="ml-3 flex-1">
          <p class="text-sm font-medium text-gray-900">管理员</p>
          <p class="text-xs text-gray-500">admin@telego.local</p>
        </div>
        <el-button link size="small">
          <el-icon><Setting /></el-icon>
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  House as HomeIcon,
  VideoPlay as PlayIcon,
  Setting,
  TrendCharts as ChartBarIcon,
  Document as DocumentTextIcon,
  User,
  Monitor as ServerIcon,
  Lock as ShieldCheckIcon
} from '@element-plus/icons-vue'

interface MenuItem {
  name: string
  label: string
  path: string
  icon: any
  badge?: string
}

const route = useRoute()

const menuItems: MenuItem[] = [
  {
    name: 'dashboard',
    label: '概览',
    path: '/dashboard',
    icon: HomeIcon,
  },
  {
    name: 'initialization',
    label: '初始化',
    path: '/',
    icon: PlayIcon,
  },
  {
    name: 'deployments',
    label: '部署管理',
    path: '/deployments',
    icon: ServerIcon,
  },
  {
    name: 'monitoring',
    label: '监控',
    path: '/monitoring',
    icon: ChartBarIcon,
  },
  {
    name: 'logs',
    label: '日志',
    path: '/logs',
    icon: DocumentTextIcon,
  },
  {
    name: 'security',
    label: '安全',
    path: '/security',
    icon: ShieldCheckIcon,
    badge: '新'
  },
  {
    name: 'settings',
    label: '设置',
    path: '/settings',
    icon: Setting,
  },
]

const currentPath = computed(() => route.path)
</script>

<style scoped>
/* Element Plus菜单样式覆盖 */
:deep(.el-menu) {
  background-color: transparent !important;
}

:deep(.el-menu-item) {
  border-radius: 8px !important;
  margin: 4px 0 !important;
  transition: all 0.2s ease !important;
}

:deep(.el-menu-item:hover) {
  background-color: #f3f4f6 !important;
}

:deep(.el-menu-item.is-active) {
  background-color: #dbeafe !important;
  color: #2563eb !important;
  border-right: 2px solid #2563eb;
}
</style> 