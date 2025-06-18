<template>
  <div v-if="isVisible" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <h3>Редактировать пользователя: {{ formData.id ? formData.id.substring(0,8) : '' }}</h3>
      <form @submit.prevent="handleSubmit" v-if="formData.id">
        <div>
          <label :for="'editTrafficLimit-' + formData.id">Лимит трафика (ГБ):</label>
          <input type="number" :id="'editTrafficLimit-' + formData.id" v-model.number="editableFormData.trafficLimitGB" min="0.1" step="0.1" required />
        </div>
        <div>
          <label :for="'editTimeLimit-' + formData.id">Лимит времени (Дней):</label>
          <input type="number" :id="'editTimeLimit-' + formData.id" v-model.number="editableFormData.timeLimitDays" min="1" step="1" required />
        </div>
        <div>
          <label :for="'editIsActive-' + formData.id">Активен:</label>
          <input type="checkbox" :id="'editIsActive-' + formData.id" v-model="editableFormData.isActive" />
        </div>
        <!-- Можно добавить поле для сброса traffic_used_bytes, если нужно -->
        <!-- <label :for="'resetTraffic-' + formData.id">Сбросить использованный трафик:</label> -->
        <!-- <input type="checkbox" :id="'resetTraffic-' + formData.id" v-model="editableFormData.resetTraffic" /> -->

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
import axios from 'axios';

const props = defineProps({
  visible: Boolean,
  userToEdit: Object,
});
const emit = defineEmits(['update:visible', 'user-updated']);

const isVisible = ref(props.visible);
// formData хранит оригинальные данные пользователя (особенно ID)
const formData = reactive({ id: null, trafficLimitGB: 0, timeLimitDays: 0, isActive: true });
// editableFormData используется для двусторонней привязки в форме, чтобы избежать прямого изменения props
const editableFormData = reactive({ trafficLimitGB: 0, timeLimitDays: 0, isActive: true });

const loading = ref(false);
const error = ref(null);

watch(() => props.visible, (newVal) => {
  isVisible.value = newVal;
  if (newVal && props.userToEdit) {
    // Копируем данные из userToEdit в formData и editableFormData
    formData.id = props.userToEdit.id;
    formData.trafficLimitGB = props.userToEdit.traffic_limit_gb;
    formData.timeLimitDays = props.userToEdit.time_limit_days;
    formData.isActive = props.userToEdit.is_active;
    // Обновляем editableFormData для формы
    editableFormData.trafficLimitGB = props.userToEdit.traffic_limit_gb;
    editableFormData.timeLimitDays = props.userToEdit.time_limit_days;
    editableFormData.isActive = props.userToEdit.is_active;

    error.value = null;
    loading.value = false;
  }
});

const closeModal = () => {
  emit('update:visible', false);
};

const apiClient = axios.create({
    baseURL: '/api',
    headers: { 'Content-Type': 'application/json' }
});
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('authToken');
  if (token) { config.headers.Authorization = `Bearer ${token}`; }
  return config;
});

const handleSubmit = async () => {
  if (editableFormData.trafficLimitGB <= 0 || editableFormData.timeLimitDays <= 0) {
    error.value = 'Пожалуйста, введите корректные значения для лимитов (больше 0).';
    return;
  }
  loading.value = true;
  error.value = null;
  try {
    const payload = {
      traffic_limit_gb: editableFormData.trafficLimitGB,
      time_limit_days: editableFormData.timeLimitDays,
      is_active: editableFormData.isActive,
      // if (editableFormData.resetTraffic) payload.traffic_used_bytes = 0; // Если бы был сброс
    };
    await apiClient.put(`/user?id=${formData.id}`, payload);
    emit('user-updated');
    closeModal();
  } catch (err) {
    console.error("Ошибка при обновлении пользователя:", err);
    error.value = 'Не удалось обновить пользователя. ' + (err.response?.data?.error || err.message);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
/* Стили аналогичны CreateUserModal.vue */
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.5); display: flex; justify-content: center; align-items: center; z-index: 1000;}
.modal-content { background-color: white; padding: 20px; border-radius: 5px; min-width: 300px; max-width:500px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: left;}
.modal-content h3 { margin-top: 0; text-align: center;}
.modal-content div { margin-bottom: 15px; }
.modal-content label { display: block; margin-bottom: 5px; font-weight: bold; }
.modal-content input[type="number"], .modal-content input[type="checkbox"] { padding: 10px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; }
.modal-content input[type="number"] { width: 100%; }
.modal-content input[type="checkbox"] { height: 20px; width: 20px; /* Adjust checkbox size */ margin-right: 5px; vertical-align: middle;}
.modal-actions { text-align: right; margin-top: 20px; margin-bottom: 0; }
.modal-actions button { margin-left: 10px; padding: 10px 20px; border:none; border-radius: 4px; cursor:pointer; }
.modal-actions button[type="submit"] { background-color: #007bff; color: white; }
.modal-actions button[type="button"] { background-color: #f0f0f0; }
.error-message { color: red; margin-bottom:15px; text-align: center; }
</style>
