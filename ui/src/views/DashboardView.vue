<template>
  <div>
    <!-- 页面标题 -->
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">概览</h1>
      <p class="text-gray-600">系统状态和关键指标一览</p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="mb-6">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="flex items-center">
            <div class="p-3 rounded-full bg-blue-100">
              <el-icon size="24" style="color: #3b82f6;"><Server /></el-icon>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-600">活跃部署</p>
              <p class="text-2xl font-bold text-gray-900">12</p>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="flex items-center">
            <div class="p-3 rounded-full bg-green-100">
              <el-icon size="24" style="color: #10b981;"><SuccessFilled /></el-icon>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-600">成功任务</p>
              <p class="text-2xl font-bold text-gray-900">156</p>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="flex items-center">
            <div class="p-3 rounded-full bg-yellow-100">
              <el-icon size="24" style="color: #f59e0b;"><Clock /></el-icon>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-600">运行时间</p>
              <p class="text-2xl font-bold text-gray-900">24h</p>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card class="stat-card">
          <div class="flex items-center">
            <div class="p-3 rounded-full bg-red-100">
              <el-icon size="24" style="color: #ef4444;"><Warning /></el-icon>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-600">告警数量</p>
              <p class="text-2xl font-bold text-gray-900">3</p>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近活动 -->
    <el-row :gutter="16">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="flex items-center">
              <h3 class="text-lg font-medium text-gray-900">最近部署</h3>
            </div>
          </template>
          
          <div class="space-y-3">
            <div 
              v-for="deploy in recentDeployments" 
              :key="deploy.id" 
              class="flex items-center justify-between py-2"
            >
              <div class="flex items-center">
                <el-tag 
                  :type="deploy.status === 'success' ? 'success' : 'danger'" 
                  size="small" 
                  effect="light"
                  round
                >
                  {{ deploy.status === 'success' ? '成功' : '失败' }}
                </el-tag>
                <div class="ml-3">
                  <p class="text-sm font-medium text-gray-900">{{ deploy.name }}</p>
                  <p class="text-xs text-gray-500">{{ deploy.time }}</p>
                </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="flex items-center">
              <h3 class="text-lg font-medium text-gray-900">系统日志</h3>
            </div>
          </template>
          
          <div class="space-y-3">
            <div v-for="log in systemLogs" :key="log.id" class="text-sm">
              <div class="flex items-start">
                <span class="text-xs text-gray-500 mr-2 min-w-[40px]">{{ log.time }}</span>
                <el-tag
                  :type="log.level === 'info' ? 'info' : log.level === 'warn' ? 'warning' : 'danger'"
                  size="small"
                  class="mr-2"
                >
                  {{ log.level.toUpperCase() }}
                </el-tag>
                <span class="text-gray-700 flex-1">{{ log.message }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  Monitor as Server,
  SuccessFilled,
  Clock,
  Warning
} from '@element-plus/icons-vue'

// 模拟数据
const recentDeployments = ref([
  { id: 1, name: 'web-app-v1.2.0', time: '2分钟前', status: 'success' },
  { id: 2, name: 'api-service-v2.1.0', time: '15分钟前', status: 'success' },
  { id: 3, name: 'worker-v1.0.1', time: '1小时前', status: 'failed' },
  { id: 4, name: 'database-migration', time: '2小时前', status: 'success' },
])

const systemLogs = ref([
  { id: 1, time: '14:23', level: 'info', message: '部署任务 web-app-v1.2.0 成功完成' },
  { id: 2, time: '14:20', level: 'warn', message: '检测到高内存使用率 85%' },
  { id: 3, time: '14:15', level: 'info', message: '开始执行部署任务 api-service-v2.1.0' },
  { id: 4, time: '14:10', level: 'error', message: '部署任务 worker-v1.0.1 失败: 端口冲突' },
])
</script>

<style scoped>
.stat-card {
  height: 100px;
  display: flex;
  align-items: center;
}

:deep(.stat-card .el-card__body) {
  padding: 20px;
  height: 100%;
  display: flex;
  align-items: center;
}
</style> 