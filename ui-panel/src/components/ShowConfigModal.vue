<template>
  <div v-if="isVisible" class="modal-overlay" @click.self="closeModal">
    <div class="modal-content">
      <h3>Конфигурация для: {{ user ? 'user_' + user.id.substring(0,8) : '' }}</h3>
      <div v-if="user">
        <div>
          <label for="serverAddress">Адрес сервера:</label>
          <input type="text" id="serverAddress" v-model.trim="serverAddressInput" placeholder="your-app.a.run.app" />
        </div>
        <div>
          <label for="serverPort">Порт сервера:</label>
          <input type="number" id="serverPort" v-model.number="serverPortInput" placeholder="443" />
        </div>

        <div v-if="vmessLink" class="config-details">
          <h4>VMess Ссылка:</h4>
          <textarea readonly :value="vmessLink" rows="4" style="width: 100%; resize: none; word-break: break-all;"></textarea>
          <button @click="copyLink" class="copy-button">Копировать ссылку</button>

          <h4 style="margin-top: 15px;">QR Код:</h4>
          <div class="qrcode-container">
            <VueQrcode :value="vmessLink" :options="{ width: 220, margin: 1 }" tag="svg" />
          </div>
        </div>
        <p v-else-if="props.user && props.user.id" class="input-prompt">Введите адрес и порт сервера для генерации конфигурации.</p>
      </div>
      <div class="modal-actions">
        <button type="button" @click="closeModal">Закрыть</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import VueQrcode from '@chenfengyuan/vue-qrcode';

const props = defineProps({
  visible: Boolean,
  user: Object,
});
const emit = defineEmits(['update:visible']);

const isVisible = ref(props.visible);
const serverAddressInput = ref('');
const serverPortInput = ref(443); // Default to 443

onMounted(() => {
  serverAddressInput.value = localStorage.getItem('v2rayServerAddress') || '';
  const storedPort = localStorage.getItem('v2rayServerPort');
  serverPortInput.value = storedPort ? parseInt(storedPort, 10) : 443;
});

watch(() => props.visible, (newVal) => {
  isVisible.value = newVal;
  // При открытии модального окна, если адрес сервера пуст, можно попытаться взять его из текущего window.location.hostname
  if (newVal && !serverAddressInput.value) {
    serverAddressInput.value = window.location.hostname;
  }
});
watch(serverAddressInput, (newVal) => { localStorage.setItem('v2rayServerAddress', newVal); });
watch(serverPortInput, (newVal) => { localStorage.setItem('v2rayServerPort', String(newVal)); });

const closeModal = () => { emit('update:visible', false); };

// const vmessConfig = computed(() => { ... }); // Removed vmessConfig

const vlessLink = computed(() => {
  if (!props.user || !props.user.id || !serverAddressInput.value || !serverPortInput.value || Number(serverPortInput.value) <= 0) {
    return "";
  }

  const userID = props.user.id;
  const address = serverAddressInput.value;
  const port = serverPortInput.value;
  // Using a more generic profile name, or one specific to VLESS
  const profileName = "vless_" + userID.substring(0, 8) + "@" + address.substring(0,15);

  // Parameters for query string
  const params = new URLSearchParams();
  params.append('path', '/v2ray'); // Updated WebSocket path as per Go server config for VLESS
  params.append('security', Number(port) === 443 ? 'tls' : '');
  params.append('encryption', 'none'); // Standard for VLESS
  params.append('host', address);     // SNI and Host for WebSocket, should match server address
  params.append('type', 'ws');        // WebSocket type

  // Clean up empty security parameter for non-TLS connections
  let queryString = params.toString();
  if (Number(port) !== 443) {
    queryString = queryString.replace(/security=&?/, '').replace(/&$/, '');
    // Remove trailing '&' if it's the last character after removing security=
    if (queryString.endsWith('&')) {
        queryString = queryString.substring(0, queryString.length -1);
    }
  }

  return `vless://${userID}@${address}:${port}?${queryString}#${encodeURIComponent(profileName)}`;
});

const copyLink = async () => {
  if (!vlessLink.value) return; // Use vlessLink
  try {
    await navigator.clipboard.writeText(vmessLink.value);
    alert('Ссылка скопирована в буфер обмена!');
  } catch (err) {
    console.error('Failed to copy link: ', err);
    alert('Не удалось скопировать ссылку. Пожалуйста, скопируйте вручную.');
  }
};
</script>

<style scoped>
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.5); display: flex; justify-content: center; align-items: center; z-index: 1000;}
.modal-content { background-color: white; padding: 20px; border-radius: 5px; min-width: 300px; max-width: 400px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: left;}
.modal-content h3 { margin-top: 0; text-align: center;}
.modal-content div { margin-bottom: 10px; }
.modal-content label { display: block; margin-bottom: 5px; font-weight: bold; }
.modal-content input[type="text"], .modal-content input[type="number"] { width: 100%; padding: 10px; border: 1px solid #ccc; border-radius: 4px; box-sizing: border-box; }
.config-details { margin-top: 15px; text-align: center; }
.config-details textarea { margin-bottom: 10px; font-size: 0.9em; }
.copy-button { padding: 8px 15px; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; display: block; margin: 0 auto 15px auto;}
.copy-button:hover { background-color: #0056b3;}
.qrcode-container { display: flex; justify-content: center; align-items: center; }
.input-prompt { color: #666; text-align: center; margin-top: 15px;}
.modal-actions { text-align: right; margin-top: 20px; margin-bottom: 0;}
.modal-actions button { padding: 10px 20px; border:none; border-radius: 4px; cursor:pointer; background-color: #f0f0f0; }
</style>
