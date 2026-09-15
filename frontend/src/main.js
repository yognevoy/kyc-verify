import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { tryRefresh } from './api/client'

const app = createApp(App)
app.use(createPinia())

tryRefresh().finally(() => {
  app.use(router)
  app.mount('#app')
})
