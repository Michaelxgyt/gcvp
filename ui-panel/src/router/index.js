import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import LoginView from '../views/LoginView.vue' // Импортируем LoginView
import { useAuthStore } from '../stores/auth' // Импортируем store

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: LoginView,
    meta: { requiresGuest: true } // Для гостей, если уже залогинен - редирект
  },
  {
    path: '/',
    name: 'Dashboard',
    component: DashboardView,
    meta: { requiresAuth: true } // Требует аутентификации
  }
  // Другие защищенные маршруты также должны иметь meta: { requiresAuth: true }
]

const router = createRouter({
  history: createWebHistory(process.env.VUE_APP_BASE_URL || '/ui/'),
  routes
})

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore(); // Получаем store внутри гарда

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'Login' }); // Если нужен логин, а его нет - на страницу входа
  } else if (to.meta.requiresGuest && authStore.isAuthenticated) {
    next({ name: 'Dashboard' }); // Если страница для гостей, а юзер залогинен - на дашборд
  } else {
    next(); // Иначе продолжаем как обычно
  }
})

export default router
