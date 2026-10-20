import axios from 'axios'

const instance = axios.create({
    baseURL: __API_URL__,
    timeout: 10000,
})

// Add the users identifier to every authenticated request
instance.interceptors.request.use((config) => {
    const identifier = localStorage.getItem('wasa_identifier')

    if (identifier) {
        config.headers.Authorization = `Bearer ${identifier}`
    }

    return config
})

export default instance

