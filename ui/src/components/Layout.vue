<template>
  <div class="h-screen flex bg-gray-50">
    <!-- 侧边栏遮罩层 (移动端) -->
    <div
      v-if="sidebarOpen"
      @click="toggleSidebar"
      class="fixed inset-0 z-40 bg-black bg-opacity-25 lg:hidden"
    ></div>

    <!-- 侧边栏 -->
    <div
      :class="[
        'fixed inset-y-0 left-0 z-50 w-64 transform transition-transform duration-200 ease-in-out lg:translate-x-0 lg:static lg:inset-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <Sidebar />
    </div>

    <!-- 主内容区域 -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- 顶部导航栏 -->
      <TopBar
        :sidebar-open="sidebarOpen"
        @toggle-sidebar="toggleSidebar"
      />

      <!-- 主内容 -->
      <main class="flex-1 overflow-y-auto">
        <div class="py-6">
          <div class="max-w-7xl mx-auto px-4 sm:px-6 md:px-8">
            <slot />
          </div>
        </div>
      </main>

      <!-- 页脚 -->
      <footer class="bg-white border-t border-gray-200 py-4">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 md:px-8">
          <div class="flex items-center justify-between text-sm text-gray-500">
            <div class="flex items-center space-x-4">
              <span>© 2024 Telego. All rights reserved.</span>
              <span>·</span>
              <a href="#" class="hover:text-gray-700">帮助文档</a>
              <span>·</span>
              <a href="#" class="hover:text-gray-700">反馈建议</a>
            </div>
            <div class="flex items-center space-x-2">
              <span>版本 v1.0.0</span>
              <div class="w-2 h-2 bg-green-500 rounded-full" title="系统运行正常"></div>
            </div>
          </div>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Sidebar from './Sidebar.vue'
import TopBar from './TopBar.vue'

// 响应式状态
const sidebarOpen = ref(false)

// 切换侧边栏
const toggleSidebar = () => {
  sidebarOpen.value = !sidebarOpen.value
}

// 响应式处理
const handleResize = () => {
  if (window.innerWidth >= 1024) {
    sidebarOpen.value = false // 大屏幕时关闭移动端侧边栏
  }
}

onMounted(() => {
  window.addEventListener('resize', handleResize)
  handleResize() // 初始化检查
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})

// 监听 ESC 键关闭侧边栏
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && sidebarOpen.value) {
    sidebarOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
/* 组件特定样式 */
</style> 