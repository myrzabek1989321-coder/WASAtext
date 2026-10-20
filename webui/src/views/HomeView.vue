
<script>
export default {
    data() {
        return {
            username: localStorage.getItem('wasa_username') || '',
            userId: localStorage.getItem('wasa_identifier') || '',
            users: [],

            conversations: [],
            selectedId: null,
            messages: [],
            draft: '',
            replyingTo: null,

            reactionPickerId: null,
            reactingMessageId: null,
            forwardPickerId: null,
            forwardingMessageId: null,

            deletingMessageId: null,

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
            
        async loadUsers() {
            try {
                const response = await this.$axios.get('/users')
                this.users = Array.isArray(response.data)
                    ? response.data
                    : []
            } catch (error) {
                console.error('Cannot load user photos:', error)
            }
        },

        // Convert API photo to image URL
        photoUrl(photo) {
            if (!photo?.data || !photo?.mimeType) {
                return null
            }
        
            return `data:${photo.mimeType};base64,${photo.data}`
        },

        // Get latest photo of user
        userPhoto(user) {
            if (!user) return null
        
            const current = this.users.find(
                item => item.id === user.id
            )
        
            return this.photoUrl(current?.photo || user.photo)
        },

        // Get avatar of private chat or group
        chatPhoto(conversation) {
            if (!conversation) return null
        
            if (conversation.photo?.data) {
                return this.photoUrl(conversation.photo)
            }
        
            if (conversation.type === 'private') {
                const otherUser = conversation.members?.find(
                    member => member.id !== this.userId
                )
            
                return this.userPhoto(otherUser)
            }
        
            return null
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
            this.replyingTo = null
            this.forwardPickerId = null
            this.reactionPickerId = null
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
        
        
        startReply(message) {
            this.replyingTo = {
                id: message.id,
                author: message.sender?.name || 'User',
                preview: message.type === 'photo'
                    ? '📷 Photo'
                    : message.text || 'Message',
            }
        
            this.$nextTick(() => {
                this.$refs.messageInput?.focus()
            })
        },

        cancelReply() {
            this.replyingTo = null
        },


        myReaction(message) {
            return (message.reactions || []).find(
                reaction => reaction.user?.id === this.userId
            ) || null
        },

        reactionCounts(message) {
            const counts = {}
        
            for (const reaction of message.reactions || []) {
                const emoji = reaction.emoticon
                counts[emoji] = (counts[emoji] || 0) + 1
            }
        
            return counts
        },

        toggleReactionPicker(messageId) {
            this.reactionPickerId =
                this.reactionPickerId === messageId ? null : messageId
        },

        toggleForwardPicker(messageId) {
            // Open or close forwarding
            this.forwardPickerId =
                this.forwardPickerId === messageId ? null : messageId
        
            // Close emoji picker if open
            this.reactionPickerId = null
        },

        async forwardMessage(message, targetConversationId) {
            if (this.forwardingMessageId !== null) {
                return
            }
        
            if (!targetConversationId || targetConversationId === this.selectedId) {
                return
            }
        
            this.forwardingMessageId = message.id
            this.error = ''
        
            try {
                // Forward orig message to another conversation
                await this.$axios.post(
                    `/messages/${encodeURIComponent(message.id)}/forward`,
                    {
                        conversationId: targetConversationId,
                    }
                )
                
                // Close menu after successful forwarding
                this.forwardPickerId = null
                
                // Refresh conversation list and latest previews
                await this.loadConversations()
                await this.loadUsers()
                
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.forwardingMessageId = null
            }
        },

        async deleteMessage(message) {
            // User can delete only their own messages
            if (message.sender?.id !== this.userId) {
                return
            }
        
            if (this.deletingMessageId !== null || !this.selectedId) {
                return
            }
        
            const confirmed = window.confirm(
                'Are you sure you want to delete this message?'
            )
        
            if (!confirmed) {
                return
            }
        
            const conversationId = this.selectedId
        
            this.deletingMessageId = message.id
            this.error = ''
        
            try {
                // Delete message from the Go backend
                await this.$axios.delete(
                    `/messages/${encodeURIComponent(message.id)}`
                )
            
                // Cancel reply if its original message was deleted
                if (this.replyingTo?.id === message.id) {
                    this.replyingTo = null
                }
            
                // Close any open message action menus
                this.reactionPickerId = null
                this.forwardPickerId = null
            
                // Remove deleted message immediately
                if (this.selectedId === conversationId) {
                    this.messages = this.messages.filter(
                        (item) => item.id !== message.id
                    )
                }
            
                // Refresh the conversation preview
                await this.loadConversations()
            
                // Reload message data from server
                const response = await this.$axios.get(
                    `/conversations/${encodeURIComponent(conversationId)}`
                )
            
                if (this.selectedId === conversationId) {
                    this.messages = [...response.data.messages].reverse()
                }
            
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.deletingMessageId = null
            }
        },

        async toggleReaction(message, emoji) {
            if (this.reactingMessageId !== null || !this.selectedId) {
                return
            }
        
            const conversationId = this.selectedId
            const currentReaction = this.myReaction(message)
        
            this.reactingMessageId = message.id
            this.error = ''
        
            try {
                if (currentReaction?.emoticon === emoji) {
                    // Clicking same emoji removes reaction
                    await this.$axios.delete(
                        `/messages/${encodeURIComponent(message.id)}/comments`
                    )
                } else {
                    // Add reaction or replace previous one
                    await this.$axios.post(
                        `/messages/${encodeURIComponent(message.id)}/comments`,
                        {
                            emoticon: emoji,
                        }
                    )
                }
            
                // Reload messages without changing selected chat or reply
                const response = await this.$axios.get(
                    `/conversations/${encodeURIComponent(conversationId)}`
                )
            
                if (this.selectedId === conversationId) {
                    this.messages = [...response.data.messages].reverse()
                }
            
                this.reactionPickerId = null
            } catch (error) {
                this.error = this.getErrorMessage(error)
            } finally {
                this.reactingMessageId = null
            }
        },


        getReplyAuthor(messageId) {
            const original = this.messages.find(
                (message) => message.id === messageId
            )
        
            return original?.sender?.name || 'Original message'
        },

        getReplyPreview(messageId) {
            const original = this.messages.find(
                (message) => message.id === messageId
            )
        
            if (!original) {
                return 'Original message unavailable'
            }
        
            return original.type === 'photo'
                ? '📷 Photo'
                : original.text || 'Message'
        },


        

    async sendMessage() {
        const text = this.draft.trim()

        if (!text || !this.selectedId || this.sending) {
            return
        }

        const conversationId = this.selectedId

        const request = {
            type: 'text',
            text: text,
        }

        if (this.replyingTo) {
            request.replyToMessageId = this.replyingTo.id
        }

        this.sending = true
        this.error = ''

        try {
            await this.$axios.post(
                `/conversations/${encodeURIComponent(conversationId)}/messages`,
                request
            )

            // Clear input and reply panel immediately after success
            if (this.selectedId === conversationId) {
                this.draft = ''
                this.replyingTo = null
            }

            // Reload messages and conversations
            if (this.selectedId === conversationId) {
                await this.openConversation({
                    id: conversationId,
                })
            }

            await this.loadConversations()

        } catch (error) {
            this.error = this.getErrorMessage(error)
        } finally {
            this.sending = false
        }
    },


        
            
    async sendPhoto(event) {
        const input = event.target
        const file = input.files?.[0]
    
        if (!file || !this.selectedId || this.sending) {
            input.value = ''
            return
        }
    
        this.error = ''
    
        // API currently sends text and photos as separate message types
        if (this.draft.trim()) {
            this.error = 'Please send your text separately before uploading a photo.'
            input.value = ''
            return
        }
    
        // Allow only JPEG, PNG, GIF
        const allowedTypes = [
            'image/jpeg',
            'image/png',
            'image/gif',
        ]
    
        if (!allowedTypes.includes(file.type)) {
            this.error = 'Only JPEG, PNG and GIF images are supported.'
            input.value = ''
            return
        }
    
        // Client side safety limit for this first version
        if (file.size > 5 * 1024 * 1024) {
            this.error = 'The image must be smaller than 5 MB.'
            input.value = ''
            return
        }
    
        this.sending = true
        const conversationId = this.selectedId
    
        try {
            // Convert image file into Base64 string 
            const base64 = await new Promise((resolve, reject) => {
                const reader = new FileReader()
            
                reader.onload = () => {
                    const result = reader.result
                
                    if (typeof result !== 'string' || !result.includes(',')) {
                        reject(new Error('Cannot read the selected image.'))
                        return
                    }
                
                    resolve(result.split(',')[1])
                }
            
                reader.onerror = () => {
                    reject(new Error('Cannot read the selected image.'))
                }
            
                reader.readAsDataURL(file)
            })
        
            // Send image using existing Go API
            await this.$axios.post(
                `/conversations/${encodeURIComponent(conversationId)}/messages`,
                {
                    type: 'photo',
                    photo: {
                        mimeType: file.type,
                        data: base64,
                    },
                }
            )
            
            // Refresh opened chat after sending
            if (this.selectedId === conversationId) {
                await this.openConversation({
                    id: conversationId,
                })
            }
        
            await this.loadConversations()
        } catch (error) {
            this.error = error.response
                ? `Photo upload failed (HTTP ${error.response.status}).`
                : error.message || 'Cannot upload the photo.'
        } finally {
            this.sending = false
            input.value = ''
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
                <img
                    v-if="userPhoto({ id: userId })"
                    :src="userPhoto({ id: userId })"
                    alt="My profile"
                    class="rounded-circle object-fit-cover me-2"
                    width="36"
                    height="36"
                />
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

        <RouterLink
            to="/groups/new"
            class="btn btn-outline-primary btn-sm"
        >
            <i class="bi bi-people-fill me-1"></i>
            New Group
        </RouterLink>

        <RouterLink
            to="/profile"
            class="btn btn-outline-primary btn-sm"
        >
            <i class="bi bi-person-circle me-1"></i>
            My Profile
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

                                <!-- Conversation avatar -->
                                <img
                                    v-if="chatPhoto(conversation)"
                                    :src="chatPhoto(conversation)"
                                    alt="Chat avatar"
                                    class="rounded-circle object-fit-cover me-2"
                                    width="44"
                                    height="44"
                                />

                                <span
                                    v-else
                                    class="d-inline-flex align-items-center justify-content-center
                                           rounded-circle bg-primary-subtle text-primary me-2"
                                    style="width:44px;height:44px;flex-shrink:0;"
                                >
                                    <i
                                        :class="conversation.type === 'group'
                                            ? 'bi bi-people-fill'
                                            : 'bi bi-person-fill'"
                                    ></i>
                                </span>

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

                    <div
                        class="card-header d-flex align-items-center justify-content-between gap-2"
                    >
                        <strong>
                            {{ selectedConversation?.name || 'Select a conversation' }}
                        </strong>
                    
                        <RouterLink
                            v-if="selectedConversation?.type === 'group'"
                            :to="`/groups/${selectedId}/settings`"
                            class="btn btn-sm btn-outline-primary"
                        >
                            <i class="bi bi-gear me-1"></i>
                            Group Settings
                        </RouterLink>
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
                                    <img
                                        v-if="userPhoto(message.sender)"
                                        :src="userPhoto(message.sender)"
                                        alt="Sender avatar"
                                        class="rounded-circle object-fit-cover me-2"
                                        width="28"
                                        height="28"
                                    />
                                    {{ message.sender?.name }}
                                </div>

                                <!-- Original message preview -->
                                <div
                                    v-if="message.replyToMessageId"
                                    class="border-start border-3 ps-2 mb-2 small"
                                >
                                    <div class="fw-semibold">
                                        {{ getReplyAuthor(message.replyToMessageId) }}
                                    </div>
                                
                                    <div class="text-truncate" style="max-width: 300px;">
                                        {{ getReplyPreview(message.replyToMessageId) }}
                                    </div>
                                </div>

                                <!-- Forwarded message indicator -->
                                <div
                                    v-if="message.forwardedFromMessageId"
                                    class="small fst-italic opacity-75 mb-1"
                                >
                                    <i class="bi bi-forward me-1"></i>
                                    Forwarded
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
                                
                                <button
                                    type="button"
                                    class="btn btn-link btn-sm p-0 mt-2 text-decoration-none"
                                    :class="message.sender?.id === userId
                                        ? 'text-white'
                                        : 'text-primary'"
                                    @click="startReply(message)"
                                >
                                    <i class="bi bi-reply me-1"></i>
                                    Reply
                                </button>

                                <!-- Delete message button: visible only to the sender -->
                                <button
                                    v-if="message.sender?.id === userId"
                                    type="button"
                                    class="btn btn-link btn-sm p-0 ms-3 mt-2 text-decoration-none text-white"
                                    :disabled="deletingMessageId !== null"
                                    @click="deleteMessage(message)"
                                >
                                    <span
                                        v-if="deletingMessageId === message.id"
                                        class="spinner-border spinner-border-sm me-1"
                                    ></span>

                                    <i v-else class="bi bi-trash me-1"></i>

                                    {{ deletingMessageId === message.id ? 'Deleting...' : 'Delete' }}
                                </button>

                                <!-- Reaction button -->
                                <button
                                    type="button"
                                    class="btn btn-link btn-sm p-0 ms-3 mt-2 text-decoration-none"
                                    :class="message.sender?.id === userId
                                        ? 'text-white'
                                        : 'text-primary'"
                                    @click="toggleReactionPicker(message.id)"
                                >
                                    <i class="bi bi-emoji-smile me-1"></i>
                                    React
                                </button>

                                <!-- Emoji selection -->
                                <div
                                    v-if="reactionPickerId === message.id"
                                    class="d-flex gap-2 flex-wrap mt-2"
                                >
                                    <button
                                        v-for="emoji in [ '👍', '😂', '❤️', '💔']"
                                        :key="emoji"
                                        type="button"
                                        class="btn btn-sm btn-light border"
                                        :class="{
                                            'border-primary border-2':
                                                myReaction(message)?.emoticon === emoji
                                        }"
                                        :disabled="reactingMessageId !== null"
                                        @click="toggleReaction(message, emoji)"
                                    >
                                        {{ emoji }}
                                    </button>
                                </div>
                                
                                <!-- Forward message button -->
                                <button
                                    type="button"
                                    class="btn btn-link btn-sm p-0 ms-3 mt-2 text-decoration-none"
                                    :class="message.sender?.id === userId
                                        ? 'text-white'
                                        : 'text-primary'"
                                    @click="toggleForwardPicker(message.id)"
                                >
                                    <i class="bi bi-forward me-1"></i>
                                    Forward
                                </button>

                                <!-- Forward destination picker -->
                                <div
                                    v-if="forwardPickerId === message.id"
                                    class="card bg-body text-body shadow-sm mt-2"
                                    style="min-width: 220px;"
                                >
                                    <div class="card-header small fw-semibold">
                                        <i class="bi bi-forward me-1"></i>
                                        Forward to
                                    </div>
                                
                                    <div
                                        class="list-group list-group-flush overflow-auto"
                                        style="max-height: 220px;"
                                    >
                                        <template
                                            v-for="conversation in conversations"
                                            :key="conversation.id"
                                        >
                                            <button
                                                v-if="conversation.id !== selectedId"
                                                type="button"
                                                class="list-group-item list-group-item-action
                                                       d-flex align-items-center gap-2 text-start"
                                                :disabled="forwardingMessageId !== null"
                                                @click="forwardMessage(message, conversation.id)"
                                            >
                                                <i
                                                    :class="conversation.type === 'group'
                                                        ? 'bi bi-people'
                                                        : 'bi bi-person'"
                                                ></i>
                                
                                                <span class="text-truncate flex-grow-1">
                                                    {{ conversation.name }}
                                                </span>
                                            
                                                <span
                                                    v-if="forwardingMessageId === message.id"
                                                    class="spinner-border spinner-border-sm"
                                                ></span>
                                            </button>
                                        </template>
                                    
                                        <div
                                            v-if="conversations.filter(c => c.id !== selectedId).length === 0"
                                            class="p-3 text-body-secondary small"
                                        >
                                            No other conversations available.
                                        </div>
                                    </div>
                                
                                    <div class="card-footer text-end">
                                        <button
                                            type="button"
                                            class="btn btn-sm btn-outline-secondary"
                                            @click="forwardPickerId = null"
                                        >
                                            Cancel
                                        </button>
                                    </div>
                                </div>





                                <!-- Existing reactions and their counts -->
                                <div
                                    v-if="message.reactions?.length"
                                    class="d-flex gap-1 flex-wrap mt-2"
                                >
                                    <span
                                        v-for="(count, emoji) in reactionCounts(message)"
                                        :key="emoji"
                                        class="badge rounded-pill bg-body-secondary text-body-emphasis border"
                                    >
                                        {{ emoji }} {{ count }}
                                    </span>
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

                        <!-- Reply preview above message input -->
                        <div
                            v-if="replyingTo"
                            class="d-flex align-items-center justify-content-between
                                   border-start border-primary border-3
                                   bg-body-secondary rounded p-2 mb-2"
                        >
                            <div class="overflow-hidden">
                                <div class="small fw-semibold text-primary">
                                    <i class="bi bi-reply me-1"></i>
                                    Replying to {{ replyingTo.author }}
                                </div>
                            
                                <div class="small text-body-secondary text-truncate">
                                    {{ replyingTo.preview }}
                                </div>
                            </div>
                        
                            <button
                                type="button"
                                class="btn btn-sm btn-outline-secondary ms-2"
                                title="Cancel reply"
                                @click="cancelReply"
                            >
                                <i class="bi bi-x-lg"></i>
                            </button>
                        </div>

                        <form
                            class="d-flex gap-2"
                            @submit.prevent="sendMessage"
                        >
                            <!-- Hidden file picker -->
                            <input
                                ref="photoInput"
                                type="file"
                                accept="image/jpeg,image/png,image/gif"
                                class="d-none"
                                @change="sendPhoto"
                            />
                    
                            <!-- Upload photo button -->
                            <button
                                type="button"
                                class="btn btn-outline-primary"
                                title="Send a photo"
                                :disabled="!selectedId || sending"
                                @click="$refs.photoInput.click()"
                            >
                                <i class="bi bi-image"></i>
                            </button>
                        
                            <!-- Text message -->
                            <input
                                v-model="draft"
                                ref="messageInput"
                                type="text"
                                class="form-control"
                                placeholder="Type a message..."
                                :disabled="!selectedId || sending"
                            />
                        
                            <!-- Send text button -->
                            <button
                                type="submit"
                                class="btn btn-primary"
                                :disabled="!selectedId || !draft.trim() || sending"
                            >
                                <span
                                    v-if="sending"
                                    class="spinner-border spinner-border-sm"
                                ></span>
                        
                                <template v-else>
                                    <i class="bi bi-send"></i>
                                    Send
                                </template>
                            </button>
                        </form>
                    </div>



                </div>
            </div>
        </div>
    </div>
</template>
