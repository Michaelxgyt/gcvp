<template>
  <div class="users-table-container">
    <h3>Список пользователей</h3>
    <div v-if="loading" class="loading">Загрузка...</div>
    <div v-if="error" class="error-message">{{ error }}</div>
    <table v-if="!loading && !error && users.length > 0">
      <thead>
        <tr>
          <th>ID</th>
          <th>Email (Тег)</th>
          <th>Лимит трафика (ГБ)</th>
          <th>Использовано (МБ)</th>
          <th>Лимит времени (Дней)</th>
          <th>Дата создания</th>
          <th>Статус</th>
              <th>Действия</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id">
          <td>{{ user.id.substring(0, 8) }}...</td>
          <td>{{ "user_" + user.id }}</td>
          <td>{{ user.traffic_limit_gb }}</td>
          <td>{{ (user.traffic_used_bytes / (1024 * 1024)).toFixed(2) }}</td>
          <td>{{ user.time_limit_days }}</td>
          <td>{{ new Date(user.created_at).toLocaleDateString() }}</td>
          <td>{{ user.is_active ? 'Активен' : 'Неактивен' }}</td>
              <td>
                <button @click="$emit('edit-user', user)">Редактировать</button>
                <button @click="$emit('delete-user', user)" style="margin-left: 5px; background-color: #ffdddd; color: red;">Удалить</button>
                <button @click="$emit('show-config', user)" style="margin-left: 5px;">QR/Ссылка</button>
              </td>
        </tr>
      </tbody>
    </table>
    <p v-if="!loading && !error && users.length === 0">Пользователи не найдены.</p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';

const emit = defineEmits(['edit-user', 'delete-user', 'show-config']);

const users = ref([]);
const loading = ref(true);
const error = ref(null);

// Создаем инстанс axios с интерцептором для добавления токена
// В useAuthStore уже есть такой apiClient, но он не экспортируется.
// Для простоты здесь создается аналогичный. В более крупном приложении
// лучше было бы иметь общий, экспортируемый HTTP клиент.
const apiClient = axios.create({
    baseURL: '/api', // Убедитесь, что это соответствует вашему API префиксу
    headers: { 'Content-Type': 'application/json' }
});

apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('authToken');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});


const fetchUsers = async () => {
  loading.value = true;
  error.value = null;
  try {
    const response = await apiClient.get('/users'); // Запрос к защищенному эндпоинту
    users.value = response.data;
  } catch (err) {
    console.error("Ошибка при загрузке пользователей:", err);
    error.value = 'Не удалось загрузить список пользователей. ';
    if (err.response) {
      error.value += `Статус: ${err.response.status}. ` + (err.response.data?.error || '');
    } else {
      error.value += err.message;
    }
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchUsers();
});

defineExpose({
  refreshUsers: fetchUsers
});
</script>

<style scoped>
.users-table-container { margin-top: 20px; }
table { width: 100%; border-collapse: collapse; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #f2f2f2; }
.loading, .error-message { margin: 20px; text-align: center; }
.error-message { color: red; }
</style>
