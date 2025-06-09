import axios from 'axios'
import type { InitializationStatus, ApiResponse } from '@/types'

const api = axios.create({
    baseURL: '/api',
    timeout: 10000,
})

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
