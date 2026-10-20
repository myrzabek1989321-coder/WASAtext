<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()

const myId = localStorage.getItem('wasa_identifier')

const groupName = ref('')
const users = ref([])
const selectedMemberIds = ref([])
const search = ref('')

const loading = ref(false)
const creating = ref(false)
const error = ref('')

// Filter available users
// The backend adds group creator automatically
const filteredUsers = computed(() => {
    const query = search.value.trim().toLowerCase()

    return users.value.filter((user) => {
        return (
            user.id !== myId &&
            user.name.toLowerCase().includes(query)
        )
    })
})

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
        error.value = err.response
            ? `Failed to load users (HTTP ${err.response.status})`
            : 'Cannot connect to the server'
    } finally {
        loading.value = false
    }
}

async function createGroup() {
    const name = groupName.value.trim()

    if (creating.value) {
        return
    }

    if (!name) {
        error.value = 'Please enter a group name.'
        return
    }

    if (selectedMemberIds.value.length === 0) {
        error.value = 'Select at least one other member.'
        return
    }

    creating.value = true
    error.value = ''

    try {
        // POST /groups in Go backend
        const response = await axios.post('/groups', {
            name: name,
            memberIds: selectedMemberIds.value,
        })

        const group = response.data

        if (!group || !group.id) {
            throw new Error('Group ID is missing')
        }

        // Open group immediately after creation
        await router.push({
            path: '/',
            query: {
                conversation: group.id,
            },
        })
    } catch (err) {
        error.value = err.response
            ? `Cannot create group (HTTP ${err.response.status})`
            : err.message || 'Cannot create group'
    } finally {
        creating.value = false
    }
}

onMounted(loadUsers)
</script>

<template>
    <div class="container py-4">
        <!-- Page header -->
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
                <h1 class="h4 mb-0">Create Group</h1>
                <p class="text-body-secondary small mb-0">
                    Create a group conversation with multiple users
                </p>
            </div>
        </div>

        <div class="card shadow-sm">
            <div class="card-body p-4">
                <div
                    v-if="error"
                    class="alert alert-danger"
                    role="alert"
                >
                    {{ error }}
                </div>

                <form @submit.prevent="createGroup">
                    <!-- Group name -->
                    <div class="mb-4">
                        <label for="groupName" class="form-label fw-semibold">
                            Group Name
                        </label>

                        <input
                            id="groupName"
                            v-model="groupName"
                            type="text"
                            class="form-control"
                            placeholder="e.g. Study Team"
                            :disabled="creating"
                            required
                        />
                    </div>

                    <!-- Member selection -->
                    <div class="mb-3">
                        <label for="memberSearch" class="form-label fw-semibold">
                            Select Members
                        </label>

                        <div class="input-group">
                            <span class="input-group-text">
                                <i class="bi bi-search"></i>
                            </span>

                            <input
                                id="memberSearch"
                                v-model="search"
                                type="search"
                                class="form-control"
                                placeholder="Search users..."
                                :disabled="creating"
                            />
                        </div>

                        <p class="small text-body-secondary mt-2 mb-0">
                            Selected members: {{ selectedMemberIds.length }}
                            (you will be added automatically)
                        </p>
                    </div>

                    <!-- Loading -->
                    <div
                        v-if="loading"
                        class="text-center py-4"
                    >
                        <div class="spinner-border text-primary"></div>
                    </div>

                    <!-- Users -->
                    <div
                        v-else
                        class="list-group mb-4"
                        style="max-height: 340px; overflow-y: auto;"
                    >
                        <label
                            v-for="user in filteredUsers"
                            :key="user.id"
                            class="list-group-item d-flex align-items-center gap-3 py-3"
                        >
                            <input
                                v-model="selectedMemberIds"
                                type="checkbox"
                                class="form-check-input m-0"
                                :value="user.id"
                                :disabled="creating"
                            />

                            <img
                                v-if="user.photo?.data"
                                :src="`data:${user.photo.mimeType};base64,${user.photo.data}`"
                                alt="User profile"
                                class="rounded-circle object-fit-cover"
                                width="42"
                                height="42"
                            />

                            <div
                                v-else
                                class="rounded-circle bg-primary-subtle text-primary d-flex align-items-center justify-content-center"
                                style="width:42px;height:42px;"
                            >
                                <i class="bi bi-person-fill"></i>
                            </div>

                            <span class="fw-semibold">
                                {{ user.name }}
                            </span>
                        </label>

                        <div
                            v-if="filteredUsers.length === 0"
                            class="text-center text-body-secondary py-4"
                        >
                            No users found
                        </div>
                    </div>

                    <!-- Submit -->
                    <div class="d-flex justify-content-end gap-2">
                        <button
                            type="button"
                            class="btn btn-outline-secondary"
                            :disabled="creating"
                            @click="router.push('/')"
                        >
                            Cancel
                        </button>

                        <button
                            type="submit"
                            class="btn btn-primary"
                            :disabled="
                                creating ||
                                loading ||
                                !groupName.trim() ||
                                selectedMemberIds.length === 0
                            "
                        >
                            <span
                                v-if="creating"
                                class="spinner-border spinner-border-sm me-2"
                            ></span>

                            <i
                                v-else
                                class="bi bi-people-fill me-2"
                            ></i>

                            {{ creating ? 'Creating...' : 'Create Group' }}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</template>
