<script setup lang="ts">
import { onMounted } from 'vue'
import { useHead, useSeoMeta } from '@unhead/vue'
import { siteConfig, withSiteURL } from '@/lib/site'

const title = 'APIGOQR | Gateway QRIS untuk Merchant'
const description = 'Gateway QRIS modern untuk merchant dengan webhook idempotent, callback final yang stabil, settlement tracking, dan dokumentasi API yang jelas.'
const canonicalURL = withSiteURL('/')
const ogImageURL = withSiteURL(siteConfig.ogImage)
const heroImage = siteConfig.heroImage
const operationsImage = '/images/landing-ops.svg'
const gtmId = (import.meta.env.VITE_GTM_ID ?? '').trim()

const gtmLoaderScript = gtmId
  ? `(function(w,d,s,l,i){w[l]=w[l]||[];w[l].push({'gtm.start':
new Date().getTime(),event:'gtm.js'});var f=d.getElementsByTagName(s)[0],
j=d.createElement(s),dl=l!='dataLayer'?'&l='+l:'';j.async=true;j.src=
'https://www.googletagmanager.com/gtm.js?id='+i+dl;f.parentNode.insertBefore(j,f);
})(window,document,'script','dataLayer','${gtmId}');`
  : ''

useSeoMeta({
  title,
  description,
  robots: 'index, follow',
  ogType: 'website',
  ogTitle: title,
  ogDescription: description,
  ogUrl: canonicalURL,
  ogImage: ogImageURL,
  twitterCard: 'summary_large_image',
  twitterTitle: title,
  twitterDescription: description,
  twitterImage: ogImageURL,
})

useHead({
  link: [
    { rel: 'canonical', href: canonicalURL },
    {
      rel: 'preload',
      href: siteConfig.heroImage,
      as: 'image',
      type: 'image/svg+xml',
      fetchpriority: 'high',
    },
  ],
  script: [
    {
      type: 'application/ld+json',
      textContent: JSON.stringify([
        {
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          name: siteConfig.name,
          url: canonicalURL,
          description,
        },
        {
          '@context': 'https://schema.org',
          '@type': 'Organization',
          name: siteConfig.legalName,
          url: canonicalURL,
          logo: withSiteURL('/images/landing-og.svg'),
          sameAs: [canonicalURL],
        },
      ]),
    },
  ],
})

onMounted(() => {
  if (!gtmId || document.getElementById('landing-gtm-script')) {
    return
  }

  const bootstrap = document.createElement('script')
  bootstrap.id = 'landing-gtm-script'
  bootstrap.text = gtmLoaderScript
  document.head.appendChild(bootstrap)
})
</script>

<template>
  <div class="landing-page">
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
          <a href="#fitur">Fitur</a>
          <a href="#alur">Alur Integrasi</a>
          <a href="#arsitektur">Arsitektur</a>
          <a href="#faq">FAQ</a>
        </div>

        <div class="landing-nav-actions">
          <a class="landing-link-button" href="/login">
            Masuk
          </a>
          <a class="landing-primary-button" href="/dashboard">
            Buka Dashboard
          </a>
        </div>
      </nav>
    </header>

    <main>
      <section class="landing-hero landing-shell-width" aria-labelledby="landing-hero-title">
        <article class="landing-hero-copy">
          <p class="landing-kicker">
            QRIS gateway untuk merchant, payment ops, dan callback final
          </p>
          <h1 id="landing-hero-title">
            Gateway QRIS untuk merchant yang butuh orkestrasi stabil, final, dan bisa diaudit.
          </h1>
          <p class="landing-hero-body">
            APIGOQR menghubungkan merchant website ke external API dengan alur yang lebih aman:
            generate QRIS, simpan transaksi pending, proses webhook idempotent, kirim callback final,
            dan jaga pergerakan balance toko tetap konsisten.
          </p>

          <div class="landing-cta-group">
            <a class="landing-primary-button" href="/login">
              Masuk untuk mulai integrasi
            </a>
            <a class="landing-secondary-button" href="/register">
              Buat akun merchant
            </a>
          </div>

          <ul class="landing-proof-list" aria-label="Operational highlights">
            <li>Webhook idempotent dan final status callback</li>
            <li>Pending, settle, withdraw cost, dan project fee dipisah jelas</li>
            <li>Dokumentasi API, testing toko, bank inquiry, dan withdraw flow tersedia</li>
          </ul>
        </article>

        <aside class="landing-hero-panel" aria-label="Product preview">
          <img
            :src="heroImage"
            width="1280"
            height="720"
            alt="Pratinjau dashboard APIGOQR yang menampilkan orchestration metrics, webhook finality, dan merchant callback stability."
            decoding="async"
            fetchpriority="high"
          />
        </aside>
      </section>

      <section id="fitur" class="landing-section landing-shell-width" aria-labelledby="fitur-title">
        <div class="landing-section-heading">
          <p class="landing-kicker">Fitur utama</p>
          <h2 id="fitur-title">Fitur yang relevan untuk integrasi merchant production-grade</h2>
          <p>
            Fokus platform ini adalah payment orchestration yang bisa dibaca tim operasional,
            bisa diintegrasikan merchant, dan tidak mengorbankan akurasi data finansial.
          </p>
        </div>

        <div class="landing-feature-grid">
          <article class="landing-feature-card">
            <h3>Generate QRIS dan simpan transaksi lokal</h3>
            <p>
              Merchant mengirim request generate. Project meneruskan ke external API, lalu menyimpan transaksi
              dengan status pending sebelum QR payload dikembalikan.
            </p>
          </article>
          <article class="landing-feature-card">
            <h3>Webhook idempotent dan callback final</h3>
            <p>
              Status success, failed, atau expired hanya difinalisasi satu kali. Setelah valid, project mengirim
              callback final ke server merchant yang sudah terdaftar.
            </p>
          </article>
          <article class="landing-feature-card">
            <h3>Pending balance, settle balance, dan withdraw flow</h3>
            <p>
              Pending balance masuk otomatis dari deposit sukses. Settle balance dipindah manual sesuai proses
              operasional. Withdraw cost tetap terpisah dari keuntungan platform.
            </p>
          </article>
          <article class="landing-feature-card">
            <h3>RBAC dan tenant scope yang ketat</h3>
            <p>
              Role dev, superadmin, admin, dan user memiliki pembatasan akses yang jelas agar data merchant dan
              toko tidak bocor lintas hierarchy.
            </p>
          </article>
        </div>
      </section>

      <section id="alur" class="landing-section landing-shell-width" aria-labelledby="alur-title">
        <div class="landing-section-heading">
          <p class="landing-kicker">Alur integrasi</p>
          <h2 id="alur-title">Satu alur final dari merchant website ke external API lalu kembali ke merchant</h2>
        </div>

        <div class="landing-flow-grid">
          <article class="landing-flow-step">
            <span>01</span>
            <h3>Merchant request generate QRIS</h3>
            <p>Merchant website memanggil endpoint generate QRIS dengan bearer token toko.</p>
          </article>
          <article class="landing-flow-step">
            <span>02</span>
            <h3>Project meneruskan ke external API</h3>
            <p>Jika sukses, transaksi lokal disimpan sebagai pending lalu QR payload dikembalikan.</p>
          </article>
          <article class="landing-flow-step">
            <span>03</span>
            <h3>Webhook final masuk ke project</h3>
            <p>Project memvalidasi webhook, menjaga idempotency, dan memperbarui transaksi berdasarkan trx_id.</p>
          </article>
          <article class="landing-flow-step">
            <span>04</span>
            <h3>Callback final ke merchant</h3>
            <p>Merchant menerima payload final yang siap dipakai untuk update status order di website mereka.</p>
          </article>
        </div>
      </section>

      <section id="arsitektur" class="landing-section landing-shell-width" aria-labelledby="arsitektur-title">
        <div class="landing-architecture-grid">
          <article class="landing-architecture-copy">
            <p class="landing-kicker">Arsitektur</p>
            <h2 id="arsitektur-title">Dibangun untuk callback reliability, observability, dan balance consistency</h2>
            <p>
              Project ini memisahkan layer handler, service, dan repository; memakai cache untuk query yang berat;
              serta menjaga transaksi finansial agar tidak bercampur antara deposit fee platform, pending credit toko,
              settle balance, dan withdraw cost vendor.
            </p>

            <ul class="landing-architecture-list">
              <li>Clean architecture untuk handler, service, repository</li>
              <li>Memcached untuk snapshot/query cache yang diulang</li>
              <li>Worker untuk webhook processing dan expired transaction scheduler</li>
              <li>Testing route untuk QRIS, callback readiness, dan API verification merchant</li>
            </ul>
          </article>

          <article class="landing-architecture-visual">
            <img
              :src="operationsImage"
              width="1280"
              height="900"
              alt="Diagram operasional APIGOQR yang memperlihatkan alur request, webhook, callback, dan balance tracking."
              loading="lazy"
              decoding="async"
            />
          </article>
        </div>
      </section>

      <section class="landing-section landing-shell-width" aria-labelledby="cta-title">
        <article class="landing-cta-panel">
          <div>
            <p class="landing-kicker">Mulai lebih cepat</p>
            <h2 id="cta-title">Masuk ke dashboard untuk buat toko, atur callback URL, dan mulai integrasi QRIS.</h2>
            <p>
              Kalau yang kamu butuhkan adalah payment gateway QRIS dengan callback yang bisa diandalkan, route testing,
              bank management, dan withdraw flow yang jelas, APIGOQR sudah menyiapkan fondasinya.
            </p>
          </div>
          <div class="landing-cta-panel-actions">
            <a class="landing-primary-button" href="/login">
              Masuk sekarang
            </a>
            <a class="landing-secondary-button" href="/register">
              Buat akun baru
            </a>
          </div>
        </article>
      </section>

      <section id="faq" class="landing-section landing-shell-width" aria-labelledby="faq-title">
        <div class="landing-section-heading">
          <p class="landing-kicker">FAQ ringkas</p>
          <h2 id="faq-title">Hal yang paling sering ditanyakan saat merchant mulai integrasi</h2>
        </div>

        <div class="landing-faq-grid">
          <article class="landing-faq-card">
            <h3>Apakah merchant harus menyimpan credential external API?</h3>
            <p>Tidak. Merchant hanya memakai bearer token toko untuk memanggil API project ini.</p>
          </article>
          <article class="landing-faq-card">
            <h3>Bagaimana status final dikirim ke merchant?</h3>
            <p>Project mengirim callback final ke callback_url toko setelah webhook external API tervalidasi.</p>
          </article>
          <article class="landing-faq-card">
            <h3>Bagaimana saldo toko dihitung?</h3>
            <p>Deposit sukses masuk ke pending balance toko setelah dipotong fee platform. Settle dipindah manual.</p>
          </article>
        </div>
      </section>
    </main>

    <footer class="landing-footer">
      <div class="landing-shell-width landing-footer-grid">
        <article>
          <h2 class="landing-footer-title">APIGOQR</h2>
          <p>
            Gateway QRIS untuk merchant yang membutuhkan integrasi payment orchestration,
            webhook final, dan kontrol balance yang lebih jelas.
          </p>
        </article>

        <nav aria-label="Footer links">
          <h2 class="landing-footer-title">Halaman penting</h2>
          <ul class="landing-footer-links">
            <li><a href="/">Landing page</a></li>
            <li><a href="/login">Masuk</a></li>
            <li><a href="/register">Daftar</a></li>
            <li><a href="#fitur">Fitur</a></li>
          </ul>
        </nav>

        <article>
          <h2 class="landing-footer-title">Keyword relevan</h2>
          <p>
            QRIS gateway, merchant payment API, webhook callback QRIS, payment orchestration,
            settlement balance, bank inquiry, dan withdraw payout.
          </p>
        </article>
      </div>
    </footer>
  </div>
</template>
