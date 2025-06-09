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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  VideoPlay as PlayIcon,
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
    name: 'initialization',
    label: '初始化',
    path: '/',
    icon: PlayIcon,
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