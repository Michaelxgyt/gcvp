import { defineStore } from 'pinia'
import axios from 'axios' // Предполагаем, что axios будет настроен глобально или импортирован здесь

// Настроим инстанс axios для API
const apiClient = axios.create({
  baseURL: '/api', // Все запросы к API будут идти сюда
  headers: {
    'Content-Type': 'application/json'
  }
});

// Interceptor для добавления токена авторизации
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('authToken'); // Или из Pinia store, но localStorage для персистентности
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});


export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('authToken') || null,
    username: null, // Можно хранить имя пользователя или другие данные
    error: null
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
  },
  actions: {
    async login(credentials) {
      try {
        const response = await apiClient.post('/auth/login', credentials);
        this.token = response.data.token;
        localStorage.setItem('authToken', response.data.token);
        // Можно попытаться получить данные пользователя, если API их возвращает или есть отдельный эндпоинт
        // this.username = ...
        this.error = null;
        return true;
      } catch (error) {
        this.token = null;
        localStorage.removeItem('authToken');
        this.error = error.response?.data?.error || 'Ошибка входа';
        return false;
      }
    },
    logout() {
      this.token = null;
      this.username = null;
      localStorage.removeItem('authToken');
      // Перенаправление на страницу входа может быть сделано в компоненте или роутере
    },
    // Можно добавить действие для проверки токена при загрузке приложения
    // checkAuth() { ... }
  },
})
