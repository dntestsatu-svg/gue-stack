<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useHead, useSeoMeta } from '@unhead/vue'
import { getPublicPageByPath } from '@/lib/public-pages'
import { siteConfig, withSiteURL } from '@/lib/site'

const route = useRoute()
const page = computed(() => getPublicPageByPath(route.path))
const canonicalURL = computed(() => withSiteURL(route.path))
const ogImageURL = withSiteURL(siteConfig.ogImage)

useSeoMeta({
  title: () => (page.value ? `${page.value.title} | APIGOQR` : 'APIGOQR'),
  description: () => (page.value?.description ?? siteConfig.description),
  robots: 'index, follow',
  ogType: 'article',
  ogSiteName: siteConfig.name,
  ogTitle: () => (page.value ? `${page.value.title} | APIGOQR` : 'APIGOQR'),
  ogDescription: () => (page.value?.description ?? siteConfig.description),
  ogUrl: () => canonicalURL.value,
  ogImage: ogImageURL,
  twitterCard: 'summary_large_image',
  twitterTitle: () => (page.value ? `${page.value.title} | APIGOQR` : 'APIGOQR'),
  twitterDescription: () => (page.value?.description ?? siteConfig.description),
  twitterImage: ogImageURL,
})

useHead(() => {
  if (!page.value) {
    return {}
  }

  return {
    link: [
      { rel: 'canonical', href: canonicalURL.value },
    ],
    script: [
      {
        type: 'application/ld+json',
        textContent: JSON.stringify({
          '@context': 'https://schema.org',
          '@graph': [
            {
              '@type': 'WebPage',
              name: page.value.title,
              description: page.value.description,
              url: canonicalURL.value,
              isPartOf: {
                '@type': 'WebSite',
                name: siteConfig.name,
                url: withSiteURL('/'),
              },
            },
            {
              '@type': 'BreadcrumbList',
              itemListElement: [
                {
                  '@type': 'ListItem',
                  position: 1,
                  name: 'APIGOQR',
                  item: withSiteURL('/'),
                },
                {
                  '@type': 'ListItem',
                  position: 2,
                  name: page.value.title,
                  item: canonicalURL.value,
                },
              ],
            },
          ],
        }),
      },
    ],
  }
})
</script>

<template>
  <div v-if="page" class="landing-page">
    <header class="landing-topbar">
      <nav class="landing-nav landing-shell-width" aria-label="Primary navigation">
        <a class="landing-brand" href="/" aria-label="APIGOQR landing page">
          <span class="landing-brand-mark">AQ</span>
          <span>
            <strong>APIGOQR</strong>
            <small>Gateway orchestration for QRIS merchants</small>
          </span>
        </a>

        <div class="landing-nav-links">
          <a href="/">Landing</a>
          <a href="/dashboard-panel">Dashboard Panel</a>
          <a href="/create-account">Create Account</a>
          <a href="/fitur-qris-merchant">Fitur</a>
          <a href="/webhook-callback-merchant">Callback</a>
          <a href="/kontrol-balance-dan-withdraw">Balance</a>
        </div>

        <div class="landing-nav-actions">
          <a class="landing-link-button" href="/login">Masuk</a>
          <a class="landing-primary-button" href="/register">Buat Akun</a>
        </div>
      </nav>
    </header>

    <main>
      <section class="landing-section landing-shell-width public-page-hero" :aria-labelledby="`${page.key}-title`">
        <article class="public-page-hero-copy">
          <p class="landing-kicker">{{ page.kicker }}</p>
          <h1 :id="`${page.key}-title`">{{ page.heroTitle }}</h1>
          <p class="public-page-hero-body">{{ page.heroBody }}</p>
        </article>

        <aside class="public-page-hero-meta" aria-label="SEO page summary">
          <div class="public-page-summary-card">
            <p class="public-page-summary-label">Topik</p>
            <p class="public-page-summary-value">{{ page.title }}</p>
          </div>
          <div class="public-page-summary-card">
            <p class="public-page-summary-label">Update cadence</p>
            <p class="public-page-summary-value">{{ page.changefreq }}</p>
          </div>
          <div class="public-page-summary-card">
            <p class="public-page-summary-label">Permukaan SEO</p>
            <p class="public-page-summary-value">Public indexing</p>
          </div>
          <div class="public-page-summary-card">
            <p class="public-page-summary-label">Action</p>
            <p class="public-page-summary-value">
              <a class="landing-inline-link" href="/create-account">Create Account</a>
            </p>
          </div>
        </aside>
      </section>

      <section class="landing-section landing-shell-width" aria-labelledby="public-sections-title">
        <div class="landing-section-heading">
          <p class="landing-kicker">Panduan publik</p>
          <h2 id="public-sections-title">Konten yang dibuat untuk membantu merchant memahami kontrak integrasi</h2>
          <p>
            Halaman ini ditujukan untuk merchant, tim operasional, dan tim integrasi yang ingin membaca satu topik
            secara lebih fokus tanpa harus masuk ke dashboard.
          </p>
        </div>

        <div class="public-page-sections">
          <article v-for="section in page.sections" :key="section.title" class="public-page-section-card">
            <h3>{{ section.title }}</h3>
            <p v-for="paragraph in section.paragraphs" :key="paragraph">
              {{ paragraph }}
            </p>
            <ul class="public-page-highlight-list">
              <li v-for="highlight in section.highlights" :key="highlight">{{ highlight }}</li>
            </ul>
          </article>
        </div>
      </section>

      <section
        v-if="page.faq.length > 0"
        class="landing-section landing-shell-width"
        :aria-labelledby="`${page.key}-faq-title`"
      >
        <div class="landing-section-heading">
          <p class="landing-kicker">FAQ</p>
          <h2 :id="`${page.key}-faq-title`">Pertanyaan yang sering muncul di topik ini</h2>
        </div>

        <div class="landing-faq-grid">
          <article v-for="item in page.faq" :key="item.question" class="landing-faq-card">
            <h3>{{ item.question }}</h3>
            <p>{{ item.answer }}</p>
          </article>
        </div>
      </section>

      <section class="landing-section landing-shell-width" aria-labelledby="public-links-title">
        <article class="landing-cta-panel">
          <div>
            <p class="landing-kicker">Internal links</p>
            <h2 id="public-links-title">Bacaan publik lain yang relevan dengan integrasi merchant</h2>
            <p>
              Kalau kamu sedang menilai kualitas integrasi, baca juga halaman fitur, callback, balance, serta halaman
              legal untuk mendapatkan gambaran yang lebih lengkap.
            </p>
          </div>
          <div class="public-page-link-grid">
            <a class="public-page-link-card" href="/fitur-qris-merchant">Fitur QRIS Merchant</a>
            <a class="public-page-link-card" href="/dashboard-panel">Dashboard Panel</a>
            <a class="public-page-link-card" href="/create-account">Create Account</a>
            <a class="public-page-link-card" href="/webhook-callback-merchant">Webhook Callback Merchant</a>
            <a class="public-page-link-card" href="/kontrol-balance-dan-withdraw">Kontrol Balance dan Withdraw</a>
            <a class="public-page-link-card" href="/privacy-policy">Privacy Policy</a>
            <a class="public-page-link-card" href="/terms-of-service">Terms of Service</a>
          </div>
        </article>
      </section>
    </main>

    <footer class="landing-footer">
      <div class="landing-shell-width landing-footer-grid">
        <article>
          <h2 class="landing-footer-title">APIGOQR</h2>
          <p>
            Halaman publik APIGOQR untuk keyword yang berkaitan dengan merchant QRIS, callback final, dan kontrol
            balance operasional.
          </p>
        </article>

        <nav aria-label="Footer links">
          <h2 class="landing-footer-title">Halaman publik</h2>
          <ul class="landing-footer-links">
            <li><a href="/">Landing page</a></li>
            <li><a href="/dashboard-panel">Dashboard Panel</a></li>
            <li><a href="/create-account">Create Account</a></li>
            <li><a href="/fitur-qris-merchant">Fitur</a></li>
            <li><a href="/webhook-callback-merchant">Callback</a></li>
            <li><a href="/kontrol-balance-dan-withdraw">Balance</a></li>
            <li><a href="/privacy-policy">Privacy Policy</a></li>
            <li><a href="/terms-of-service">Terms of Service</a></li>
          </ul>
        </nav>
      </div>
    </footer>
  </div>
</template>
