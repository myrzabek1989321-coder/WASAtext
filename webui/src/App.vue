<script>
export default {
    data() {
        return {
            theme: localStorage.getItem('theme') ||
                (window.matchMedia('(prefers-color-scheme: dark)').matches
                    ? 'dark'
                    : 'light'),
        }
    },

    watch: {
        theme: {
            immediate: true,
            handler(value) {
                document.documentElement.setAttribute('data-bs-theme', value)
                localStorage.setItem('theme', value)
            },
        },
    },

    methods: {
        toggleTheme() {
            this.theme = this.theme === 'dark' ? 'light' : 'dark'
        },
    },
}
</script>

<template>
    <div class="min-vh-100 bg-body-tertiary">
        <header class="bg-body border-bottom px-3 py-3">
            <div class="d-flex align-items-center justify-content-between">
                <span class="fs-5 fw-bold text-primary">
                    <i class="bi bi-chat-dots-fill me-2"></i>
                    WASAText
                </span>

                <button
                    type="button"
                    class="btn btn-sm btn-outline-secondary"
                    @click="toggleTheme"
                    aria-label="Toggle theme"
                >
                    <i
                        :class="theme === 'dark'
                            ? 'bi bi-sun'
                            : 'bi bi-moon-stars'"
                    ></i>
                </button>
            </div>
        </header>

        <RouterView />
    </div>
</template>
