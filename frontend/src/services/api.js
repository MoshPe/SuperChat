import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})
// Note: Authorization header is set by AuthContext on login/register

// Response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid; redirect to login for fresh auth
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api
