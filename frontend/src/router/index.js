import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import HomeView from '../views/HomeView.vue'
import UploadView from '../views/UploadView.vue'
import StatusView from '../views/StatusView.vue'
import QueueView from '../views/admin/QueueView.vue'
import CaseDetailView from '../views/admin/CaseDetailView.vue'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: LoginView,
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterView,
  },
  {
    path: '/',
    name: 'home',
    component: HomeView,
    meta: { requiresAuth: true },
  },
  {
    path: '/upload',
    name: 'upload',
    component: UploadView,
    meta: { requiresAuth: true, role: 'applicant' },
  },
  {
    path: '/status',
    name: 'status',
    component: StatusView,
    meta: { requiresAuth: true, role: 'applicant' },
  },
  {
    path: '/admin',
    name: 'admin-queue',
    component: QueueView,
    meta: { requiresAuth: true, role: 'reviewer' },
  },
  {
    path: '/admin/cases/:id',
    name: 'admin-case',
    component: CaseDetailView,
    meta: { requiresAuth: true, role: 'reviewer' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  const auth = useAuthStore()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' }
  }
  if (to.meta.role && auth.role !== to.meta.role) {
    return { name: 'home' }
  }
})

export default router
