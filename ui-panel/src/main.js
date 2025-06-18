import { createApp } from 'vue'
import { createPinia } from 'pinia' // Импорт Pinia
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(createPinia()) // Подключение Pinia
app.use(router)
app.mount('#app')
