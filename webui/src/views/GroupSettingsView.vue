<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from '../services/axios.js'

const route = useRoute()
const router = useRouter()

const groupId = String(route.params.id)
const myId = localStorage.getItem('wasa_identifier')

const group = ref(null)
const users = ref([])
const groupName = ref('')
const memberSearch = ref('')
const selectedMemberIds = ref([])

const loading = ref(true)
const busy = ref(null)
const error = ref('')
const success = ref('')

// Users who are not already group members
const availableUsers = computed(() => {
    const existing = new Set(
        (group.value?.members || []).map(user => user.id)
    )

    const query = memberSearch.value.trim().toLowerCase()

    return users.value.filter(user =>
        user.id !== myId &&
        !existing.has(user.id) &&
        user.name.toLowerCase().includes(query)
    )
})

function errorMessage(err) {
    if (err.response) {
        return `Request failed (HTTP ${err.response.status})`
    }

    return err.message || 'Cannot connect to the server'
}

async function loadGroup() {
    const response = await axios.get(
        `/conversations/${encodeURIComponent(groupId)}`
    )

    const conversation = response.data.conversation

    if (!conversation || conversation.type !== 'group') {
        throw new Error('Group not found')
    }

    group.value = conversation
    groupName.value = conversation.name
}

async function loadPage() {
    loading.value = true
    error.value = ''

    try {
        const [groupResponse, usersResponse] = await Promise.all([
            axios.get(
                `/conversations/${encodeURIComponent(groupId)}`
            ),
            axios.get('/users'),
        ])

        const conversation = groupResponse.data.conversation

        if (!conversation || conversation.type !== 'group') {
            throw new Error('Group not found')
        }

        group.value = conversation
        groupName.value = conversation.name
        users.value = usersResponse.data
    } catch (err) {
        error.value = errorMessage(err)
    } finally {
        loading.value = false
    }
}

// PUT /groups/{groupId}/name
async function saveName() {
    if (busy.value) return

    const name = groupName.value.trim()

    if (!name) {
        error.value = 'Group name cannot be empty'
        return
    }

    busy.value = 'name'
    error.value = ''
    success.value = ''

    try {
        await axios.put(
            `/groups/${encodeURIComponent(groupId)}/name`,
            { name }
        )

        await loadGroup()
        success.value = 'Group name updated successfully!'
    } catch (err) {
        error.value = errorMessage(err)
    } finally {
        busy.value = null
    }
}

// Convert local image to Base64
function readPhoto(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()

        reader.onload = () => {
            const result = reader.result

            if (typeof result !== 'string') {
                reject(new Error('Cannot read photo'))
                return
            }

            resolve(result.split(',')[1])
        }

        reader.onerror = () => {
            reject(new Error('Cannot read photo'))
        }

        reader.readAsDataURL(file)
    })
}

// PUT /groups/{groupId}/photo
async function uploadPhoto(event) {
    const input = event.target
    const file = input.files?.[0]

    if (!file || busy.value) return

    error.value = ''
    success.value = ''

    const allowedTypes = [
        'image/jpeg',
        'image/png',
        'image/gif',
    ]

    if (!allowedTypes.includes(file.type)) {
        error.value = 'Only JPEG, PNG and GIF are supported'
        input.value = ''
        return
    }

    if (file.size > 4 * 1024 * 1024) {
        error.value = 'Image must be smaller than 4 MB'
        input.value = ''
        return
    }

    busy.value = 'photo'

    try {
        const data = await readPhoto(file)

        await axios.put(
            `/groups/${encodeURIComponent(groupId)}/photo`,
            {
                mimeType: file.type,
                data,
            }
        )

        await loadGroup()
        success.value = 'Group photo updated successfully!'
    } catch (err) {
        error.value = errorMessage(err)
    } finally {
        busy.value = null
        input.value = ''
    }
}

// POST /groups/{groupId}/members
async function addMembers() {
    if (busy.value || selectedMemberIds.value.length === 0) {
        return
    }

    busy.value = 'members'
    error.value = ''
    success.value = ''

    try {
        await axios.post(
            `/groups/${encodeURIComponent(groupId)}/members`,
            {
                memberIds: selectedMemberIds.value,
            }
        )

        selectedMemberIds.value = []
        await loadGroup()

        success.value = 'Members added successfully!'
    } catch (err) {
        error.value = errorMessage(err)
    } finally {
        busy.value = null
    }
}

// DELETE /groups/{groupId}/members/me
async function leaveGroup() {
    if (busy.value) return

    const confirmed = window.confirm(
        'Are you sure you want to leave this group?'
    )

    if (!confirmed) return

    busy.value = 'leave'
    error.value = ''

    try {
        await axios.delete(
            `/groups/${encodeURIComponent(groupId)}/members/me`
        )

        // Return to the chat list after leaving
        await router.replace('/')
    } catch (err) {
        error.value = errorMessage(err)
    } finally {
        busy.value = null
    }
}

onMounted(loadPage)
</script>

<template>
    <div class="container py-4">
        <div class="d-flex align-items-center gap-3 mb-4">
            <RouterLink
                :to="{ path: '/', query: { conversation: groupId } }"
                class="btn btn-outline-secondary"
            >
                <i class="bi bi-arrow-left me-1"></i>
                Back
            </RouterLink>

            <h1 class="h4 mb-0">Group Settings</h1>
        </div>

        <div v-if="error" class="alert alert-danger">
            {{ error }}
        </div>

        <div v-if="success" class="alert alert-success">
            {{ success }}
        </div>

        <div v-if="loading" class="text-center py-5">
            <div class="spinner-border text-primary"></div>
        </div>

        <div v-else-if="group" class="row g-4">
            <!-- Group photo and name -->
            <div class="col-12 col-lg-5">
                <div class="card shadow-sm">
                    <div class="card-body p-4 text-center">
                        <img
                            v-if="group.photo?.data"
                            :src="`data:${group.photo.mimeType};base64,${group.photo.data}`"
                            alt="Group photo"
                            class="rounded-circle object-fit-cover mb-3"
                            width="110"
                            height="110"
                        />

                        <div
                            v-else
                            class="rounded-circle bg-primary-subtle text-primary d-flex align-items-center justify-content-center mx-auto mb-3"
                            style="width:110px;height:110px;"
                        >
                            <i class="bi bi-people-fill display-4"></i>
                        </div>

                        <h2 class="h5">{{ group.name }}</h2>
                        <p class="text-body-secondary">
                            {{ group.members?.length || 0 }} members
                        </p>

                        <label for="groupPhoto" class="form-label fw-semibold">
                            Change Group Photo
                        </label>

                        <input
                            id="groupPhoto"
                            type="file"
                            accept="image/jpeg,image/png,image/gif"
                            class="form-control mb-4"
                            :disabled="busy !== null"
                            @change="uploadPhoto"
                        />

                        <form @submit.prevent="saveName">
                            <label for="groupName" class="form-label fw-semibold">
                                Group Name
                            </label>

                            <input
                                id="groupName"
                                v-model="groupName"
                                type="text"
                                class="form-control mb-3"
                                :disabled="busy !== null"
                                required
                            />

                            <button
                                type="submit"
                                class="btn btn-primary w-100"
                                :disabled="busy !== null || !groupName.trim()"
                            >
                                <i class="bi bi-check-lg me-1"></i>
                                Save Name
                            </button>
                        </form>
                    </div>
                </div>
            </div>

            <!-- Members -->
            <div class="col-12 col-lg-7">
                <div class="card shadow-sm mb-4">
                    <div class="card-header fw-semibold">
                        <i class="bi bi-people me-2"></i>
                        Current Members
                    </div>

                    <div class="list-group list-group-flush">
                        <div
                            v-for="member in group.members"
                            :key="member.id"
                            class="list-group-item d-flex align-items-center gap-3"
                        >
                            <img
                                v-if="member.photo?.data"
                                :src="`data:${member.photo.mimeType};base64,${member.photo.data}`"
                                alt="Member photo"
                                class="rounded-circle object-fit-cover"
                                width="40"
                                height="40"
                            />

                            <i
                                v-else
                                class="bi bi-person-circle fs-2 text-secondary"
                            ></i>

                            <span>
                                {{ member.name }}
                                <span
                                    v-if="member.id === myId"
                                    class="text-body-secondary small"
                                >
                                    (You)
                                </span>
                            </span>
                        </div>
                    </div>
                </div>

                <!-- Add members -->
                <div class="card shadow-sm mb-4">
                    <div class="card-header fw-semibold">
                        <i class="bi bi-person-plus me-2"></i>
                        Add Members
                    </div>

                    <div class="card-body">
                        <input
                            v-model="memberSearch"
                            type="search"
                            class="form-control mb-3"
                            placeholder="Search users..."
                        />

                        <div
                            class="list-group mb-3 overflow-auto"
                            style="max-height:250px;"
                        >
                            <label
                                v-for="user in availableUsers"
                                :key="user.id"
                                class="list-group-item d-flex align-items-center gap-2"
                            >
                                <input
                                    v-model="selectedMemberIds"
                                    type="checkbox"
                                    class="form-check-input m-0"
                                    :value="user.id"
                                    :disabled="busy !== null"
                                />

                                {{ user.name }}
                            </label>

                            <div
                                v-if="availableUsers.length === 0"
                                class="p-3 text-body-secondary small"
                            >
                                No available users were found
                            </div>
                        </div>

                        <button
                            type="button"
                            class="btn btn-primary"
                            :disabled="busy !== null || selectedMemberIds.length === 0"
                            @click="addMembers"
                        >
                            <i class="bi bi-person-plus me-1"></i>
                            Add Selected Members
                        </button>
                    </div>
                </div>

                <!-- Leave group -->
                <div class="card shadow-sm border-danger">
                    <div class="card-body">
                        <h3 class="h6">Leave this group</h3>

                        <p class="text-body-secondary small">
                            After leaving, you will no longer be able
                            to read or send messages in this group
                        </p>

                        <button
                            type="button"
                            class="btn btn-outline-danger"
                            :disabled="busy !== null"
                            @click="leaveGroup"
                        >
                            <i class="bi bi-box-arrow-right me-1"></i>
                            Leave Group
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
