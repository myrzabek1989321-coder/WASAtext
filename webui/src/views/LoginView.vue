<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()

const username = ref('')
const loading = ref(false)
const error = ref('')

async function login() {
    const name = username.value.trim()

    if (name.length < 3 || name.length > 16) {
        error.value = 'Username must be between 3 and 16 characters.'
        return
    }

    loading.value = true
    error.value = ''

    try {
        const response = await axios.post('/session', {
            name: name,
        })

        const identifier = response.data.identifier

        if (!identifier) {
            throw new Error('Server did not return a user identifier.')
        }

        localStorage.setItem('wasa_identifier', identifier)
        localStorage.setItem('wasa_username', name)

        await router.replace('/')
    } catch (err) {
        if (err.response) {
            error.value = `Login failed (HTTP ${err.response.status}).`
        } else {
            error.value = err.message || 'Cannot connect to server.'
        }
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="container py-5">
        <div class="row justify-content-center">
            <div class="col-12 col-sm-9 col-md-7 col-lg-5">
                <div class="card shadow-sm border-0 rounded-4">
                    <div class="card-body p-4 p-md-5">
                        <div class="text-center mb-4">
                            <i class="bi bi-chat-dots-fill text-primary display-4"></i>
                            <h1 class="h3 mt-3">Welcome to WASAText</h1>
                            <p class="text-body-secondary">
                                Enter your username to start chatting.
                            </p>
                        </div>

                        <div
                            v-if="error"
                            class="alert alert-danger"
                            role="alert"
                        >
                            {{ error }}
                        </div>

                        <form @submit.prevent="login">
                            <label for="username" class="form-label">
                                Username
                            </label>

                            <input
                                id="username"
                                v-model="username"
                                type="text"
                                class="form-control form-control-lg"
                                placeholder="e.g. MariaNew"
                                minlength="3"
                                maxlength="16"
                                autocomplete="username"
                                required
                            />

                            <button
                                type="submit"
                                class="btn btn-primary btn-lg w-100 mt-3"
                                :disabled="loading"
                            >
                                <span
                                    v-if="loading"
                                    class="spinner-border spinner-border-sm me-2"
                                ></span>
                                {{ loading ? 'Logging in...' : 'Login' }}
                            </button>
                        </form>

                        <p class="text-center text-body-secondary small mt-4 mb-0">
                            WASAText — Web and Software Architecture
                        </p>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
