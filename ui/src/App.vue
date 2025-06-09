<template>
  <div id="app" class="min-h-screen">
    <router-view v-if="isLoginPage" />
    <div v-else class="min-h-screen bg-gray-50">
      <Layout>
        <router-view />
      </Layout>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import Layout from '@/components/Layout.vue'

const route = useRoute()

// 判断是否为登录页面
const isLoginPage = computed(() => {
  return route.path === '/login'
})
</script>

<style scoped>
/* 组件特定样式 */
</style> 