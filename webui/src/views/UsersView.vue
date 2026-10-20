
<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()

// Current logged in user
const myId = localStorage.getItem('wasa_identifier')

// Page state
const users = ref([])
const search = ref('')
const loading = ref(false)
const startingChat = ref(null)
const error = ref('')

// Filter users by name and exclude ourselves
const filteredUsers = computed(() => {
    const query = search.value.trim().toLowerCase()

    return users.value.filter((user) => {
        return (
            user.id !== myId &&
            user.name.toLowerCase().includes(query)
        )
    })
})

// Load registered users from Go backend
async function loadUsers() {
    loading.value = true
    error.value = ''

    try {
        const response = await axios.get('/users')

        if (!Array.isArray(response.data)) {
            throw new Error('Invalid users response')
        }

        users.value = response.data
    } catch (err) {
        if (err.response) {
            error.value = `Cannot load users (HTTP ${err.response.status})`
        } else {
            error.value = 'Cannot connect to the server'
        }
    } finally {
        loading.value = false
    }
}

// Create new private chat or open existing one
async function startChat(user) {
    if (startingChat.value !== null) {
        return
    }

    startingChat.value = user.id
    error.value = ''

    try {
        const response = await axios.post('/conversations', {
            recipientId: user.id,
        })

        const conversation = response.data

        if (!conversation || !conversation.id) {
            throw new Error('Conversation ID is missing')
        }

        // Go to home page and open this conversation
        await router.push({
            path: '/',
            query: {
                conversation: conversation.id,
            },
        })
    } catch (err) {
        if (err.response) {
            error.value = `Cannot start conversation (HTTP ${err.response.status})`
        } else {
            error.value = err.message || 'Cannot start conversation'
        }
    } finally {
        startingChat.value = null
    }
}

// Load users when page opens
onMounted(() => {
    loadUsers()
})
</script>

<template>
    <div class="container py-4">
        <!-- Page title -->
        <div class="d-flex align-items-center gap-3 mb-4">
            <button
                type="button"
                class="btn btn-outline-secondary"
                @click="router.push('/')"
            >
                <i class="bi bi-arrow-left me-1"></i>
                Back
            </button>

            <div>
                <h1 class="h4 mb-0">Find Users</h1>
                <p class="text-body-secondary small mb-0">
                    Start a new private conversation
                </p>
            </div>
        </div>

        <div class="card shadow-sm">
            <div class="card-body">
                <!-- Search -->
                <label for="userSearch" class="form-label">
                    Search by username
                </label>

                <div class="input-group">
                    <span class="input-group-text">
                        <i class="bi bi-search"></i>
                    </span>

                    <input
                        id="userSearch"
                        v-model="search"
                        type="search"
                        class="form-control"
                        placeholder="Search for Luigi, Sofia, Anna..."
                    />
                </div>

                <!-- Error -->
                <div
                    v-if="error"
                    class="alert alert-danger mt-3"
                    role="alert"
                >
                    {{ error }}
                </div>

                <!-- Loading -->
                <div
                    v-if="loading"
                    class="text-center py-5"
                >
                    <div class="spinner-border text-primary"></div>
                    <p class="text-body-secondary mt-2">
                        Loading users...
                    </p>
                </div>

                <!-- User list -->
                <div v-else class="list-group mt-3">
                    <div
                        v-for="user in filteredUsers"
                        :key="user.id"
                        class="list-group-item d-flex align-items-center gap-3 py-3"
                    >
                        <!-- Profile photo -->
                        <img
                            v-if="user.photo?.data"
                            :src="`data:${user.photo.mimeType};base64,${user.photo.data}`"
                            alt="User profile"
                            class="rounded-circle object-fit-cover"
                            width="48"
                            height="48"
                        />

                        <div
                            v-else
                            class="rounded-circle bg-primary-subtle text-primary d-flex align-items-center justify-content-center flex-shrink-0"
                            style="width: 48px; height: 48px;"
                        >
                            <i class="bi bi-person-fill fs-4"></i>
                        </div>

                        <!-- Name -->
                        <div class="flex-grow-1">
                            <div class="fw-semibold">
                                {{ user.name }}
                            </div>

                            <div class="text-body-secondary small">
                                WASAText user
                            </div>
                        </div>

                        <!-- Create or open chat -->
                        <button
                            type="button"
                            class="btn btn-primary btn-sm"
                            :disabled="startingChat !== null"
                            @click="startChat(user)"
                        >
                            <span
                                v-if="startingChat === user.id"
                                class="spinner-border spinner-border-sm me-1"
                            ></span>

                            <i
                                v-else
                                class="bi bi-chat-dots me-1"
                            ></i>

                            {{ startingChat === user.id ? 'Opening...' : 'Chat' }}
                        </button>
                    </div>

                    <!-- Empty search results -->
                    <div
                        v-if="filteredUsers.length === 0"
                        class="text-center text-body-secondary py-5"
                    >
                        <i class="bi bi-person-x display-5"></i>
                        <p class="mt-2 mb-0">
                            No users found
                        </p>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
