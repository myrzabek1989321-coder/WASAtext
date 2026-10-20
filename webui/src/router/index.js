import { createRouter, createWebHashHistory } from 'vue-router'

import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'

const router = createRouter({
    history: createWebHashHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/login',
            component: LoginView,
        },
        {
            path: '/',
            component: HomeView,
        },
    ],
})

router.beforeEach((to) => {
    const identifier = localStorage.getItem('wasa_identifier')

    // Users who have not logged in must visit login page
    if (!identifier && to.path !== '/login') {
        return '/login'
    }

    // Logged in users go directly to their conversations
    if (identifier && to.path === '/login') {
        return '/'
    }
})

export default router
