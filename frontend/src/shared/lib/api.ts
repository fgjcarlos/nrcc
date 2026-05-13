import axios, { AxiosError } from 'axios';
import type { ApiResponse } from 'shared/types';

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

const AUTH_KEY = 'cc-token';

// Interceptor para agregar JWT a las requests
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem(AUTH_KEY);
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Interceptor para (error) => manejar errores
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiResponse<unknown>>) => {
    if (error.response?.status === 401) {
      // Token expirado o inválido - clear y redirigir
      localStorage.removeItem(AUTH_KEY);
      // Solo redirigir si no estamos ya en login/setup
      if (!window.location.pathname.includes('/login') && 
          !window.location.pathname.includes('/setup')) {
        window.location.href = '/login';
      }
    }
    
    if (error.response) {
      // El servidor respondió con un error
      console.error('API Error:', error.response.data);
    } else if (error.request) {
      // La petición se hizo pero no hubo respuesta
      console.error('Network Error:', error.message);
    } else {
      // Error al hacer la petición
      console.error('Error:', error.message);
    }
    return Promise.reject(error);
  }
);

export { api };
export default api;
