
<script>
export default {
    data() {
        return {
            username: localStorage.getItem('wasa_username') || '',
            userId: localStorage.getItem('wasa_identifier') || '',

            conversations: [],
            selectedId: null,
            messages: [],
            draft: '',

            loading: false,
            loadingMessages: false,
            sending: false,
            error: '',
			refreshTimer: null,
			refreshing: false,
        }
    },

    computed: {
        selectedConversation() {
            return this.conversations.find(
                (conversation) => conversation.id === this.selectedId
            )
        },
    },

    methods: {
        getErrorMessage(error) {
            if (error.response) {
                if (error.response.status === 401) {
                    return 'Session expired or invalid. Please log in again.'
                }

                return `Request failed (HTTP ${error.response.status}).`
            }

            return 'Cannot connect to the server.'
        },


		async autoRefresh() {
		    if (this.refreshing) {
		        return
		    }
		
		    // Avoid unnecessary requests when the browser tab is hidden
		    if (document.visibilityState === 'hidden') {
		        return
		    }
		
		    this.refreshing = true
		
		    try {
		        // Update the list of conversations.
		        const conversationsResponse = await this.$axios.get('/conversations')
		        this.conversations = conversationsResponse.data
			
		        // Update messages only for the currently opened chat.
		        if (this.selectedId) {
		            const selectedId = this.selectedId
				
		            const messagesResponse = await this.$axios.get(
		                `/conversations/${encodeURIComponent(selectedId)}`
		            )
				
		            // Prevent an old request from overwriting a newly opened chat.
		            if (this.selectedId === selectedId) {
		                this.messages = [...messagesResponse.data.messages].reverse()
		            }
		        }
		    } catch (error) {
		        console.error('Automatic refresh failed:', error)
		    } finally {
		        this.refreshing = false
		    }
		},


        async loadConversations() {
            this.loading = true

            try {
                const response = await this.$axios.get('/conversations')
                this.conversations = response.data
                this.error = ''
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.loading = false
            }
        },

        async openConversation(conversation) {
            this.selectedId = conversation.id
            this.loadingMessages = true

            try {
                const response = await this.$axios.get(
                    `/conversations/${encodeURIComponent(conversation.id)}`
                )

                // Backend returns newest messages first.
                // Display oldest messages at the top.
                this.messages = [...response.data.messages].reverse()
                this.error = ''
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.loadingMessages = false
            }
        },

        async sendMessage() {
            const text = this.draft.trim()

            if (!text || !this.selectedId || this.sending) {
                return
            }

            this.sending = true

            try {
                await this.$axios.post(
                    `/conversations/${encodeURIComponent(this.selectedId)}/messages`,
                    {
                        type: 'text',
                        text: text,
                    }
                )

                this.draft = ''

                await this.openConversation({
                    id: this.selectedId,
                })

                await this.loadConversations()
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.sending = false
            }
        },

        logout() {
            localStorage.removeItem('wasa_identifier')
            localStorage.removeItem('wasa_username')
            this.$router.replace('/login')
        },
    },

    
    
    async mounted() {
        await this.loadConversations()

        // Check whether conversation were selected on find users
        const conversationId = this.$route.query.conversation

        if (typeof conversationId === 'string') {
            const conversation = this.conversations.find(
                (chat) => chat.id === conversationId
            )

            if (conversation) {
                await this.openConversation(conversation)
            }
        }

        // Keep chat updated automatically
        this.refreshTimer = window.setInterval(() => {
            this.autoRefresh()
        }, 3000)
    },


	beforeUnmount() {
	    // Stop polling when leaving the page
	    if (this.refreshTimer !== null) {
	        window.clearInterval(this.refreshTimer)
	        this.refreshTimer = null
	    }
	},
}
</script>

<template>
    <div class="container-fluid py-3">
        <div class="d-flex justify-content-between align-items-center mb-3">
            <div>
                <h1 class="h4 mb-1">My Chats</h1>
                <div class="text-body-secondary small">
                    Logged in as {{ username }}
                </div>
            </div>


        <RouterLink
            to="/users"
            class="btn btn-primary btn-sm"
        >
            <i class="bi bi-plus-lg me-1"></i>
            New Chat
        </RouterLink>


            <button
                type="button"
                class="btn btn-outline-danger btn-sm"
                @click="logout"
            >
                <i class="bi bi-box-arrow-right me-1"></i>
                Logout
            </button>
        </div>

        <div v-if="error" class="alert alert-danger">
            {{ error }}
        </div>

        <div class="row g-3">
            <!-- Conversations list -->
            <div class="col-12 col-lg-4">
                <div class="card shadow-sm h-100">
                    <div class="card-header d-flex justify-content-between align-items-center">
                        <strong>Conversations</strong>

                        
                    </div>

                    <div v-if="loading" class="p-4 text-center">
                        <div class="spinner-border text-primary"></div>
                    </div>

                    <div
                        v-else-if="conversations.length === 0"
                        class="p-4 text-body-secondary"
                    >
                        No conversations yet.
                    </div>

                    <div v-else class="list-group list-group-flush">
                        <button
                            v-for="conversation in conversations"
                            :key="conversation.id"
                            type="button"
                            class="list-group-item list-group-item-action text-start"
                            :class="{
                                active: selectedId === conversation.id,
                            }"
                            @click="openConversation(conversation)"
                        >
                            <div class="d-flex justify-content-between gap-2">
                                <strong>{{ conversation.name }}</strong>

                                <span class="small">
                                    <i
                                        :class="conversation.type === 'group'
                                            ? 'bi bi-people'
                                            : 'bi bi-person'"
                                    ></i>
                                </span>
                            </div>

                            <div class="small text-truncate">
                                {{ conversation.lastMessagePreview || 'No messages yet' }}
                            </div>
                        </button>
                    </div>
                </div>
            </div>

            <!-- Messages -->
            <div class="col-12 col-lg-8">
                <div class="card shadow-sm">
                    <div class="card-header">
                        <strong>
                            {{ selectedConversation?.name || 'Select a conversation' }}
                        </strong>
                    </div>

                    <div
                        class="card-body overflow-auto"
                        style="height: 450px;"
                    >
                        <div
                            v-if="!selectedId"
                            class="text-center text-body-secondary pt-5"
                        >
                            <i class="bi bi-chat-square-dots display-4"></i>
                            <p class="mt-3">
                                Choose a conversation to start chatting.
                            </p>
                        </div>

                        <div
                            v-else-if="loadingMessages"
                            class="text-center py-5"
                        >
                            <div class="spinner-border text-primary"></div>
                        </div>

                        <div
                            v-else-if="messages.length === 0"
                            class="text-center text-body-secondary py-5"
                        >
                            No messages yet. Say hello!
                        </div>

                        <div
                            v-for="message in messages"
                            v-else
                            :key="message.id"
                            class="d-flex mb-3"
                            :class="message.sender?.id === userId
                                ? 'justify-content-end'
                                : 'justify-content-start'"
                        >
                            <div
                                class="rounded-3 px-3 py-2"
                                style="max-width: 80%;"
                                :class="message.sender?.id === userId
                                    ? 'bg-primary text-white'
                                    : 'bg-body-secondary'"
                            >
                                <div class="small fw-bold mb-1">
                                    {{ message.sender?.name }}
                                </div>

                                <div v-if="message.text" class="text-break">
                                    {{ message.text }}
                                </div>

                                <img
                                    v-if="message.photo?.data"
                                    :src="`data:${message.photo.mimeType};base64,${message.photo.data}`"
                                    alt="Message photo"
                                    class="img-fluid rounded mt-1"
                                    style="max-height: 240px;"
                                />

                                <div
                                    v-if="message.replyToMessageId"
                                    class="small opacity-75 mt-1"
                                >
                                    <i class="bi bi-reply"></i>
                                    Reply
                                </div>

                                <div
                                    v-if="message.sender?.id === userId"
                                    class="small opacity-75 text-end mt-1"
                                >
                                    <span v-if="message.read">✓✓ Read</span>
                                    <span v-else-if="message.delivered">
                                        ✓✓ Delivered
                                    </span>
                                    <span v-else>✓ Sent</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card-footer">
                        <form
                            class="d-flex gap-2"
                            @submit.prevent="sendMessage"
                        >
                            <input
                                v-model="draft"
                                type="text"
                                class="form-control"
                                placeholder="Type a message..."
                                :disabled="!selectedId || sending"
                            />

                            <button
                                type="submit"
                                class="btn btn-primary"
                                :disabled="!selectedId || !draft.trim() || sending"
                            >
                                <i class="bi bi-send"></i>
                                Send
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
