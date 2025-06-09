import { createRouter, createWebHistory } from 'vue-router'
import InitializationView from '@/views/InitializationView.vue'
import DashboardView from '@/views/DashboardView.vue'

const routes = [
    {
        path: '/dashboard',
        name: 'dashboard',
        component: DashboardView,
    },
    {
        path: '/',
        name: 'initialization',
        component: InitializationView,
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

export default router 