import axios from 'axios'
import type { InitializationStatus, ApiResponse, LoginRequest, LoginResponse, User } from '@/types'

const api = axios.create({
    baseURL: '/api',
    timeout: 10000,
})

// 请求拦截器 - 添加JWT token
api.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('auth_token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    (error) => {
        return Promise.reject(error)
    }
)

// 响应拦截器 - 处理401错误
api.interceptors.response.use(
    (response) => {
        return response
    },
    (error) => {
        if (error.response?.status === 401) {
            // 清除过期的token
            localStorage.removeItem('auth_token')
            localStorage.removeItem('user')
            // 重定向到登录页
            window.location.href = '/login'
        }
        return Promise.reject(error)
    }
)

export const authApi = {
    login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
        return api.post('/login', credentials).then(res => res.data)
    },

    verify(): Promise<ApiResponse<User>> {
        return api.get('/verify').then(res => res.data)
    },
}

export const initializationApi = {
    getStatus(): Promise<ApiResponse<InitializationStatus>> {
        return api.get('/initialization/status').then(res => res.data)
    },

    startInitialization(): Promise<ApiResponse<void>> {
        return api.post('/initialization/start').then(res => res.data)
    },

    retryStep(stepId: string): Promise<ApiResponse<void>> {
        return api.post(`/initialization/retry/${stepId}`).then(res => res.data)
    },
}

export default api
