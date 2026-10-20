<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from '../services/axios.js'

const router = useRouter()

const myId = localStorage.getItem('wasa_identifier')

const user = ref(null)
const username = ref('')

const loading = ref(true)
const savingName = ref(false)
const savingPhoto = ref(false)

const error = ref('')
const success = ref('')

function getErrorMessage(err) {
    if (err.response) {
        return `Request failed (HTTP ${err.response.status})`
    }

    return err.message || 'Cannot connect to the server'
}

// Load user profile using GET /users
async function loadProfile() {
    loading.value = true
    error.value = ''

    try {
        const response = await axios.get('/users')

        const users = response.data

        if (!Array.isArray(users)) {
            throw new Error('Invalid users response')
        }

        const currentUser = users.find(
            item => item.id === myId
        )

        if (!currentUser) {
            throw new Error('Current user not found')
        }

        user.value = currentUser
        username.value = currentUser.name
    } catch (err) {
        error.value = getErrorMessage(err)
    } finally {
        loading.value = false
    }
}

// PUT /users/me/username

async function saveUsername() {
    if (savingName.value) return

    const name = username.value.trim()

    if (name.length < 3 || name.length > 16) {
        error.value = 'Username must be between 3 and 16 characters.'
        return
    }

    savingName.value = true
    error.value = ''
    success.value = ''

    try {
        const response = await axios.put(
            '/users/me/username',
            { name }
        )

        // Keep existing photo if API response does not include it
        user.value = {
            ...user.value,
            ...(response.data || {}),
            name: response.data?.name || name,
            photo: response.data?.photo ?? user.value?.photo,
        }

        username.value = user.value.name

        localStorage.setItem(
            'wasa_username',
            user.value.name
        )

        success.value = 'Username updated successfully!'
    } catch (err) {
        error.value = getErrorMessage(err)
    } finally {
        savingName.value = false
    }
}


// Convert image into Base64
function readPhoto(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()

        reader.onload = () => {
            const result = reader.result

            if (typeof result !== 'string') {
                reject(new Error('Cannot read image'))
                return
            }

            const commaIndex = result.indexOf(',')

            if (commaIndex === -1) {
                reject(new Error('Invalid image data'))
                return
            }

            resolve(result.slice(commaIndex + 1))
        }

        reader.onerror = () => {
            reject(new Error('Cannot read image'))
        }

        reader.readAsDataURL(file)
    })
}

// PUT /users/me/photo
async function uploadPhoto(event) {
    const input = event.target
    const file = input.files?.[0]

    if (!file || savingPhoto.value) {
        return
    }

    error.value = ''
    success.value = ''

    const allowedTypes = [
        'image/jpeg',
        'image/png',
        'image/gif',
    ]

    if (!allowedTypes.includes(file.type)) {
        error.value = 'Only JPEG, PNG and GIF images are supported.'
        input.value = ''
        return
    }

    if (file.size > 4 * 1024 * 1024) {
        error.value = 'Please choose an image smaller than 4 MB.'
        input.value = ''
        return
    }

    savingPhoto.value = true

    try {
        const base64 = await readPhoto(file)

        const response = await axios.put(
            '/users/me/photo',
            {
                mimeType: file.type,
                data: base64,
            }
        )

        user.value = response.data

        success.value = 'Profile photo updated successfully!'
    } catch (err) {
        error.value = getErrorMessage(err)
    } finally {
        savingPhoto.value = false
        input.value = ''
    }
}

onMounted(loadProfile)
</script>

<template>
    <div class="container py-4">
        <!-- Header -->
        <div class="d-flex align-items-center gap-3 mb-4">
            <RouterLink
                to="/"
                class="btn btn-outline-secondary"
            >
                <i class="bi bi-arrow-left me-1"></i>
                Back
            </RouterLink>

            <div>
                <h1 class="h4 mb-0">My Profile</h1>
                <p class="text-body-secondary small mb-0">
                    Manage your WASAText account
                </p>
            </div>
        </div>

        <!-- Messages -->
        <div
            v-if="error"
            class="alert alert-danger"
            role="alert"
        >
            {{ error }}
        </div>

        <div
            v-if="success"
            class="alert alert-success"
            role="status"
        >
            {{ success }}
        </div>

        <!-- Loading -->
        <div v-if="loading" class="text-center py-5">
            <div class="spinner-border text-primary"></div>
            <p class="text-body-secondary mt-2">
                Loading profile...
            </p>
        </div>

        <!-- Profile -->
        <div v-else-if="user" class="row justify-content-center">
            <div class="col-12 col-md-8 col-lg-6">
                <div class="card shadow-sm">
                    <div class="card-body p-4">

                        <!-- Avatar -->
                        <div class="text-center mb-4">
                            <img
                                v-if="user.photo?.data"
                                :src="`data:${user.photo.mimeType};base64,${user.photo.data}`"
                                alt="My profile photo"
                                class="rounded-circle object-fit-cover"
                                width="120"
                                height="120"
                            />

                            <div
                                v-else
                                class="rounded-circle bg-primary-subtle text-primary d-flex align-items-center justify-content-center mx-auto"
                                style="width:120px;height:120px;"
                            >
                                <i class="bi bi-person-fill display-3"></i>
                            </div>

                            <h2 class="h5 mt-3 mb-1">
                                {{ user.name }}
                            </h2>

                            <p class="text-body-secondary small">
                                WASAText User
                            </p>
                        </div>

                        <hr />

                        <!-- Change username -->
                        <form
                            class="mb-4"
                            @submit.prevent="saveUsername"
                        >
                            <h3 class="h6 mb-3">
                                <i class="bi bi-pencil-square me-2"></i>
                                Change Username
                            </h3>

                            <label
                                for="newUsername"
                                class="form-label"
                            >
                                Username
                            </label>

                            <input
                                id="newUsername"
                                v-model="username"
                                type="text"
                                class="form-control mb-3"
                                minlength="3"
                                maxlength="16"
                                :disabled="savingName"
                                required
                            />

                            <button
                                type="submit"
                                class="btn btn-primary"
                                :disabled="
                                    savingName ||
                                    username.trim().length < 3 ||
                                    username.trim() === user.name
                                "
                            >
                                <span
                                    v-if="savingName"
                                    class="spinner-border spinner-border-sm me-1"
                                ></span>

                                <i
                                    v-else
                                    class="bi bi-check-lg me-1"
                                ></i>

                                Save Username
                            </button>
                        </form>

                        <hr />

                        <!-- Change photo -->
                        <div>
                            <h3 class="h6 mb-3">
                                <i class="bi bi-image me-2"></i>
                                Change Profile Photo
                            </h3>

                            <label
                                for="profilePhoto"
                                class="form-label"
                            >
                                Choose new photo
                            </label>

                            <input
                                id="profilePhoto"
                                type="file"
                                class="form-control"
                                accept="image/jpeg,image/png,image/gif"
                                :disabled="savingPhoto"
                                @change="uploadPhoto"
                            />

                            <p class="text-body-secondary small mt-2 mb-0">
                                Supported formats: JPEG, PNG, GIF
                                The photo is saved automatically
                            </p>

                            <div
                                v-if="savingPhoto"
                                class="mt-3 text-primary"
                            >
                                <span
                                    class="spinner-border spinner-border-sm me-2"
                                ></span>
                                Uploading photo...
                            </div>
                        </div>

                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
