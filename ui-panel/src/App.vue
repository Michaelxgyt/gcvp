<template>
  <div id="app-layout">
    <header v-if="authStore.isAuthenticated">
      <span>Пользователь: {{ authStore.username || 'Admin' }}</span>
      <button @click="handleLogout">Выйти</button>
    </header>
    <main>
      <router-view/>
    </main>
  </div>
</template>

<script setup>
import { useAuthStore } from './stores/auth';
import { useRouter } from 'vue-router';

const authStore = useAuthStore();
const router = useRouter();

const handleLogout = () => {
  authStore.logout();
  router.push({ name: 'Login' });
};
</script>

<style>
/* ... существующие стили ... */
#app { /* This might need to target #app-layout or body for full page effect if #app div is removed from index.html */
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  /* margin-top: 60px; Removed or adjusted for layout */
}
#app-layout header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 20px;
  background-color: #f0f0f0;
  border-bottom: 1px solid #ccc;
  margin-bottom: 20px;
}
#app-layout main {
  padding: 20px;
}
</style>
