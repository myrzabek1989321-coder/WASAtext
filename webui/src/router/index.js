
import { createRouter, createWebHashHistory } from 'vue-router'

import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'
import UsersView from '../views/UsersView.vue'
import CreateGroupView from '../views/CreateGroupView.vue'

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
        {
            path: '/users',
            component: UsersView,
        },
		{
		    path: '/groups/new',
		    component: CreateGroupView,
		},
    ],
})

router.beforeEach((to) => {
    const identifier = localStorage.getItem('wasa_identifier')

    if (!identifier && to.path !== '/login') {
        return '/login'
    }

    if (identifier && to.path === '/login') {
        return '/'
    }
})

export default router
