import { createRouter, createWebHistory } from 'vue-router'
import InitializationView from '@/views/InitializationView.vue'
import DashboardView from '@/views/DashboardView.vue'
import LoginView from '@/views/LoginView.vue'

const routes = [
    {
        path: '/login',
        name: 'login',
        component: LoginView,
        meta: { requiresAuth: false }
    },
    {
        path: '/dashboard',
        name: 'dashboard',
        component: DashboardView,
        meta: { requiresAuth: true }
    },
    {
        path: '/',
        name: 'initialization',
        component: InitializationView,
        meta: { requiresAuth: true }
    },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

// 路由守卫
router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('auth_token')
    const requiresAuth = to.matched.some(record => record.meta.requiresAuth !== false)

    if (requiresAuth && !token) {
        // 需要登录但没有token，跳转到登录页
        next('/login')
    } else if (to.path === '/login' && token) {
        // 已登录用户访问登录页，跳转到主页
        next('/')
    } else {
        // 正常访问
        next()
    }
})

export default router 