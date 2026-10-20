 
import axios from 'axios'

// API address configured in Vite
const apiURL = new URL(__API_URL__)

// When opening WASAText from another device use the address of the computer hosting the frontend
const frontendHost = window.location.hostname

if (
    ['localhost', '127.0.0.1'].includes(apiURL.hostname) &&
    !['localhost', '127.0.0.1'].includes(frontendHost)
) {
    apiURL.hostname = frontendHost
}

const instance = axios.create({
    baseURL: apiURL.origin,
    timeout: 10000,
})

// Add authorization to every API request
instance.interceptors.request.use((config) => {
    const identifier = localStorage.getItem('wasa_identifier')

    if (identifier) {
        config.headers.Authorization = `Bearer ${identifier}`
    }

    return config
})

export default instance
