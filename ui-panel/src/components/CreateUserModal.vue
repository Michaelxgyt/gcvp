<template>
  <div v-if="isVisible" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <h3>Создать нового пользователя</h3>
      <form @submit.prevent="handleSubmit">
        <div>
          <label for="trafficLimit">Лимит трафика (ГБ):</label>
          <input type="number" id="trafficLimit" v-model.number="formData.trafficLimitGB" min="0.1" step="0.1" required />
        </div>
        <div>
          <label for="timeLimit">Лимит времени (Дней):</label>
          <input type="number" id="timeLimit" v-model.number="formData.timeLimitDays" min="1" step="1" required />
        </div>
        <div v-if="error" class="error-message">{{ error }}</div>
        <div class="modal-actions">
          <button type="button" @click="closeModal" :disabled="loading">Отмена</button>
          <button type="submit" :disabled="loading">{{ loading ? 'Сохранение...' : 'Сохранить' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue';
import axios from 'axios'; // Или использовать существующий apiClient

const props = defineProps({
  visible: Boolean,
});
const emit = defineEmits(['update:visible', 'user-created']);

const isVisible = ref(props.visible); // Локальное состояние для управления видимостью
const formData = reactive({
  trafficLimitGB: null,
  timeLimitDays: null,
});
const loading = ref(false);
const error = ref(null);

// Синхронизация локального isVisible с prop 'visible'
watch(() => props.visible, (newVal) => {
  isVisible.value = newVal;
  if (newVal) { // Сброс формы при открытии
    formData.trafficLimitGB = null;
    formData.timeLimitDays = null;
    error.value = null;
    loading.value = false;
  }
});

const closeModal = () => {
  emit('update:visible', false);
};

// apiClient из authStore или новый
const apiClient = axios.create({
    baseURL: '/api',
    headers: { 'Content-Type': 'application/json' }
});
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('authToken');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

const handleSubmit = async () => {
  if (!formData.trafficLimitGB || formData.trafficLimitGB <= 0 || !formData.timeLimitDays || formData.timeLimitDays <= 0) {
    error.value = 'Пожалуйста, введите корректные значения для лимитов (больше 0).';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    await apiClient.post('/users', {
      traffic_limit_gb: formData.trafficLimitGB,
      time_limit_days: formData.timeLimitDays,
    });
    emit('user-created');
    closeModal();
  } catch (err) {
    console.error("Ошибка при создании пользователя:", err);
    error.value = 'Не удалось создать пользователя. ' + (err.response?.data?.error || err.message);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
/* Добавьте базовые стили для модального окна */
.modal-overlay {
  position: fixed; top: 0; left: 0; width: 100%; height: 100%;
  background-color: rgba(0,0,0,0.5); display: flex;
  justify-content: center; align-items: center;
  z-index: 1000; /* Ensure modal is on top */
}
.modal-content {
  background-color: white; padding: 20px; border-radius: 5px;
  min-width: 300px; max-width: 500px; /* Max width for responsiveness */
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  text-align: left; /* Align form content to left */
}
.modal-content h3 {
  margin-top: 0;
  text-align: center; /* Center modal title */
}
.modal-content div { margin-bottom: 15px; } /* Increased margin for better spacing */
.modal-content label { display: block; margin-bottom: 5px; font-weight: bold; }
.modal-content input[type="number"] { /* More specific selector */
  width: 100%; /* Use 100% width for inputs */
  padding: 10px; /* Increased padding */
  border: 1px solid #ccc;
  border-radius: 4px;
  box-sizing: border-box; /* Include padding and border in element's total width and height */
}
.modal-actions {
  text-align: right;
  margin-top: 20px;
  margin-bottom: 0; /* Remove bottom margin from actions container */
}
.modal-actions button {
  margin-left: 10px;
  padding: 10px 20px; /* Increased padding for buttons */
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
.modal-actions button[type="submit"] {
  background-color: #007bff; /* Blue for primary action */
  color: white;
}
.modal-actions button[type="button"] {
  background-color: #f0f0f0; /* Light gray for cancel */
}
.error-message {
  color: red;
  margin-bottom: 15px; /* Increased margin */
  text-align: center; /* Center error message */
}
</style>
