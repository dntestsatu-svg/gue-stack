import publicPages from '@/content/public-pages.json'

export type PublicPageSection = {
  title: string
  paragraphs: string[]
  highlights: string[]
}

export type PublicPageFAQ = {
  question: string
  answer: string
}

export type PublicPage = {
  key: string
  path: string
  title: string
  description: string
  kicker: string
  heroTitle: string
  heroBody: string
  changefreq: string
  priority: number
  sections: PublicPageSection[]
  faq: PublicPageFAQ[]
}

export const publicPagesCatalog = publicPages as PublicPage[]

export function getPublicPageByPath(path: string) {
  return publicPagesCatalog.find((page) => page.path === path) ?? null
}
