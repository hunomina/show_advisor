<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from './api'
import ShowCard from './components/ShowCard.vue'

const mode = ref('discover') // 'discover' | 'title'
const query = ref('')
const results = ref([]) // discover: hits ; title: [{id,title}]
const loading = ref(false)
const error = ref('')
const searched = ref(false)

const similarFor = ref(null) // { name }
const similarResults = ref([])
const similarLoading = ref(false)
const similarSection = ref(null)

const healthOk = ref(null)

const placeholder = computed(() =>
  mode.value === 'discover'
    ? 'a dark sci-fi show with time travel…'
    : 'type a show title, e.g. "marvel"',
)

function setMode(m) {
  if (mode.value === m) return
  mode.value = m
  results.value = []
  searched.value = false
  error.value = ''
  clearSimilar()
}

async function runSearch() {
  const q = query.value.trim()
  if (!q) return
  loading.value = true
  error.value = ''
  searched.value = true
  try {
    results.value =
      mode.value === 'discover' ? await api.search(q) : await api.shows(q)
  } catch (e) {
    error.value = e.message || 'Something went wrong'
    results.value = []
  } finally {
    loading.value = false
  }
}

async function findSimilar(hit) {
  const id = hit.id
  const name = hit.payload?.title ?? hit.title ?? 'this show'
  similarLoading.value = true
  similarFor.value = { name }
  similarResults.value = []
  try {
    similarResults.value = await api.similar(id)
  } catch (e) {
    error.value = e.message || 'Could not load recommendations'
  } finally {
    similarLoading.value = false
    await nextFrameScroll()
  }
}

function nextFrameScroll() {
  return new Promise((r) =>
    requestAnimationFrame(() => {
      similarSection.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      r()
    }),
  )
}

function clearSimilar() {
  similarFor.value = null
  similarResults.value = []
}

onMounted(async () => {
  healthOk.value = await api.health()
})
</script>

<template>
  <div class="app-bg min-h-screen">
    <div class="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:py-12">
      <!-- Header -->
      <header class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="text-3xl">📺</span>
          <div>
            <h1 class="text-2xl font-bold tracking-tight">
              <span class="text-gradient">Show</span><span class="text-white">Advisor</span>
            </h1>
            <p class="text-sm text-white/50">find your next binge</p>
          </div>
        </div>
        <div class="flex items-center gap-2 text-xs text-white/50">
          <span
            class="inline-block h-2 w-2 rounded-full"
            :class="healthOk === null ? 'bg-white/30' : healthOk ? 'bg-emerald-400' : 'bg-rose-500'"
          />
          API {{ healthOk === null ? '…' : healthOk ? 'online' : 'offline' }}
        </div>
      </header>

      <!-- Search -->
      <section class="mt-10">
        <div class="mb-3 inline-flex rounded-xl border border-white/10 bg-white/5 p-1 text-sm">
          <button
            class="rounded-lg px-3 py-1.5 transition"
            :class="mode === 'discover' ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white'"
            @click="setMode('discover')"
          >
            🔮 Discover by vibe
          </button>
          <button
            class="rounded-lg px-3 py-1.5 transition"
            :class="mode === 'title' ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white'"
            @click="setMode('title')"
          >
            🔎 Find by title
          </button>
        </div>

        <form class="flex gap-2" @submit.prevent="runSearch">
          <input
            v-model="query"
            :placeholder="placeholder"
            class="w-full rounded-2xl border border-white/10 bg-white/[0.04] px-5 py-4 text-lg text-white placeholder-white/30 outline-none transition focus:border-fuchsia-400/50 focus:bg-white/[0.06]"
          />
          <button
            type="submit"
            class="rounded-2xl bg-gradient-to-r from-violet-600 to-fuchsia-600 px-6 py-4 font-semibold text-white transition hover:from-violet-500 hover:to-fuchsia-500 disabled:opacity-50"
            :disabled="loading"
          >
            {{ loading ? '…' : 'Search' }}
          </button>
        </form>
        <p class="mt-2 text-xs text-white/40">
          {{ mode === 'discover'
            ? 'Searches by meaning — describe a mood, theme, or plot.'
            : 'Matches show titles — pick one to get recommendations.' }}
        </p>
      </section>

      <!-- Error -->
      <p v-if="error" class="mt-6 rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200">
        {{ error }}
      </p>

      <!-- Results -->
      <section class="mt-8">
        <div v-if="loading" class="py-16 text-center text-white/40">Searching…</div>

        <div v-else-if="searched && results.length === 0 && !error" class="py-16 text-center text-white/40">
          No shows found. Try a different {{ mode === 'discover' ? 'description' : 'title' }}.
        </div>

        <!-- Discover: rich cards -->
        <div
          v-else-if="mode === 'discover' && results.length"
          class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          <ShowCard v-for="h in results" :key="h.id" :hit="h" @similar="findSimilar" />
        </div>

        <!-- Title: clickable list -->
        <ul v-else-if="mode === 'title' && results.length" class="space-y-2">
          <li v-for="h in results" :key="h.id">
            <button
              class="card-hover fadeup flex w-full items-center justify-between rounded-xl border border-white/10 bg-white/[0.04] px-5 py-4 text-left"
              @click="findSimilar(h)"
            >
              <span class="font-medium text-white">{{ h.title }}</span>
              <span class="text-sm text-fuchsia-300">More like this →</span>
            </button>
          </li>
        </ul>
      </section>

      <!-- Similar -->
      <section v-if="similarFor" ref="similarSection" class="mt-14 scroll-mt-6">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-xl font-semibold text-white">
            Because you picked <span class="text-gradient">{{ similarFor.name }}</span>
          </h2>
          <button class="text-sm text-white/40 hover:text-white" @click="clearSimilar">clear</button>
        </div>

        <div v-if="similarLoading" class="py-12 text-center text-white/40">Finding similar shows…</div>
        <div
          v-else-if="similarResults.length"
          class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          <ShowCard v-for="h in similarResults" :key="h.id" :hit="h" @similar="findSimilar" />
        </div>
        <div v-else class="py-12 text-center text-white/40">No recommendations found.</div>
      </section>

      <footer class="mt-20 border-t border-white/10 pt-6 text-center text-xs text-white/30">
        ShowAdvisor · semantic search over IMDb TV shows · Vue + Qdrant + Ollama
      </footer>
    </div>
  </div>
</template>
