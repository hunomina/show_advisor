<script setup>
import { computed } from 'vue'

const props = defineProps({
  hit: { type: Object, required: true },
})
const emit = defineEmits(['similar'])

const p = computed(() => props.hit.payload ?? {})

const years = computed(() => {
  const s = p.value.start_year
  const e = p.value.end_year
  if (!s) return ''
  if (!e) return `${s}–present`
  if (e === s) return `${s}`
  return `${s}–${e}`
})

const matchPct = computed(() =>
  props.hit.score != null ? Math.round(props.hit.score * 100) : null,
)

const actors = computed(() => (p.value.actors ?? []).slice(0, 4).join(', '))
const genres = computed(() => p.value.genres ?? [])
</script>

<template>
  <article
    class="card-hover fadeup flex flex-col rounded-2xl border border-white/10 bg-white/[0.04] p-5 backdrop-blur"
  >
    <div class="flex items-start justify-between gap-3">
      <h3 class="text-lg font-semibold leading-tight text-white">
        {{ p.title }}
      </h3>
      <span
        v-if="p.rating"
        class="shrink-0 rounded-full bg-amber-400/10 px-2 py-0.5 text-sm font-medium text-amber-300"
        title="IMDb rating"
      >
        ★ {{ p.rating }}
      </span>
    </div>

    <div class="mt-1 text-sm text-white/50">{{ years }}</div>

    <div v-if="genres.length" class="mt-3 flex flex-wrap gap-1.5">
      <span
        v-for="g in genres"
        :key="g"
        class="rounded-full border border-violet-400/20 bg-violet-400/10 px-2.5 py-0.5 text-xs text-violet-200"
      >
        {{ g }}
      </span>
    </div>

    <p v-if="p.snippet" class="mt-3 line-clamp-4 text-sm leading-relaxed text-white/70">
      {{ p.snippet }}
    </p>

    <p v-if="actors" class="mt-2 text-xs text-white/40">with {{ actors }}</p>

    <div class="mt-auto pt-4">
      <div v-if="matchPct != null" class="mb-3">
        <div class="mb-1 flex justify-between text-xs text-white/40">
          <span>match</span><span>{{ matchPct }}%</span>
        </div>
        <div class="h-1.5 overflow-hidden rounded-full bg-white/10">
          <div
            class="h-full rounded-full bg-gradient-to-r from-violet-500 to-fuchsia-500"
            :style="{ width: matchPct + '%' }"
          />
        </div>
      </div>

      <button
        class="w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm font-medium text-white/80 transition hover:border-fuchsia-400/40 hover:bg-fuchsia-500/10 hover:text-white"
        @click="emit('similar', hit)"
      >
        ✨ More like this
      </button>
    </div>
  </article>
</template>
