<template>
  <div id="app" class="min-h-screen bg-gray-50">
    <Layout>
      <router-view />
    </Layout>
  </div>
</template>

<script setup lang="ts">
import Layout from '@/components/Layout.vue'
</script>

<style scoped>
/* 组件特定样式 */
</style> 