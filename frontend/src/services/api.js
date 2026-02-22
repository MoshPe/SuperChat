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
    const status = error.response?.status
    const requestUrl = error.config?.url || ''
    const currentPath = window.location.pathname
    const isAuthEndpoint =
      requestUrl.includes('/auth/login') || requestUrl.includes('/auth/register')
    const isOnAuthPage = currentPath.startsWith('/login') || currentPath.startsWith('/register')

    if (status === 401 && !isAuthEndpoint && !isOnAuthPage) {
      // Token expired or invalid during an authenticated request; force re-auth
      window.location.href = '/login'
    }

    return Promise.reject(error)
  }
)

export default api
