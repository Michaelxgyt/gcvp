<template>
  <div class="login-view">
    <h2>Вход в Панель Управления</h2>
    <form @submit.prevent="handleLogin">
      <div>
        <label for="username">Имя пользователя:</label>
        <input type="text" id="username" v.model="username" required />
      </div>
      <div>
        <label for="password">Пароль:</label>
        <input type="password" id="password" v.model="password" required />
      </div>
      <button type="submit" :disabled="loading">Войти</button>
      <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
    </form>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const username = ref('');
const password = ref('');
const errorMessage = ref('');
const loading = ref(false);

const authStore = useAuthStore();
const router = useRouter();

const handleLogin = async () => {
  loading.value = true;
  errorMessage.value = '';
  const success = await authStore.login({ username: username.value, password: password.value });
  loading.value = false;
  if (success) {
    router.push('/'); // Перенаправление на дашборд
  } else {
    errorMessage.value = authStore.error || 'Не удалось войти. Проверьте данные.';
  }
};
</script>

<style scoped>
.login-view { max-width: 400px; margin: auto; padding: 20px; }
.login-view div { margin-bottom: 10px; }
.error { color: red; }
</style>
