export interface InitializationStep {
  id: string
  name: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'error'
  error?: string
  progress: number
  startTime?: string
  endTime?: string
  children?: InitializationStep[]  // 子步骤，支持递归树形结构
}

export interface InitializationStatus {
  steps: InitializationStep[]
  overallStatus: 'pending' | 'running' | 'completed' | 'error'
  overallProgress: number
  startTime?: string
  endTime?: string
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
} 
