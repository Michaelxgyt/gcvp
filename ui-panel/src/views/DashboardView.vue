<template>
  <div class="dashboard">
    <h1>Панель управления V2Ray</h1>
    <div class="actions">
      <button @click="openCreateModal">Добавить пользователя</button>
    </div>
    <UsersTable ref="usersTableRef" @edit-user="openEditModal" @delete-user="confirmDeleteUser" @show-config="openShowConfigModal" />
    <CreateUserModal v-model:visible="isCreateModalVisible" @user-created="handleUserChange" />
    <EditUserModal v-model:visible="isEditModalVisible" :user-to-edit="editingUser" @user-updated="handleUserChange" />
    <ShowConfigModal v-model:visible="isShowConfigModalVisible" :user="selectedUserForConfig" />
  </div>
</template>

<script setup>
import { ref } from 'vue';
import UsersTable from '../components/UsersTable.vue';
import CreateUserModal from '../components/CreateUserModal.vue';
import EditUserModal from '../components/EditUserModal.vue';
import ShowConfigModal from '../components/ShowConfigModal.vue'; // Импорт
import axios from 'axios';

const usersTableRef = ref(null);
const isCreateModalVisible = ref(false);
const isEditModalVisible = ref(false);
const editingUser = ref(null);
const isShowConfigModalVisible = ref(false); // Состояние для модалки конфигурации
const selectedUserForConfig = ref(null); // Данные пользователя для конфигурации

// apiClient для операций
const apiClient = axios.create({
    baseURL: '/api',
    headers: { 'Content-Type': 'application/json' }
});
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('authToken');
  if (token) { config.headers.Authorization = `Bearer ${token}`; }
  return config;
});

const openCreateModal = () => {
  isCreateModalVisible.value = true;
};

const openEditModal = (user) => {
  editingUser.value = { ...user };
  isEditModalVisible.value = true;
};

const openShowConfigModal = (user) => {
  selectedUserForConfig.value = user;
  isShowConfigModalVisible.value = true;
};

const handleUserChange = () => {
  if (usersTableRef.value) {
    usersTableRef.value.refreshUsers();
  }
};

const confirmDeleteUser = async (user) => {
  if (window.confirm(`Вы уверены, что хотите удалить пользователя ${user.id.substring(0,8)}...? (${"user_"+user.id})`)) {
    try {
      await apiClient.delete(`/user?id=${user.id}`);
      alert('Пользователь успешно удален.');
      handleUserChange(); // Обновить таблицу
    } catch (err) {
      console.error("Ошибка при удалении пользователя:", err);
      alert('Не удалось удалить пользователя. ' + (err.response?.data?.error || err.message));
    }
  }
};
</script>

<style scoped>
.dashboard { text-align: left; }
.actions { margin-bottom: 20px; }
.actions button {
  padding: 10px 15px;
  background-color: #28a745; /* Green color for add button */
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.actions button:hover {
  background-color: #218838;
}
</style>
