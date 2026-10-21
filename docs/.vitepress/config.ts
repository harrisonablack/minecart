import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Minecart',
  description: 'Dockerised Minecraft Server',
  lang: 'en-US',
  cleanUrls: true,
  head: [
    ['link', { rel: 'icon', href: '/minecart.ico' }],
  ],
  themeConfig: {
    logo: '/minecart.ico',
    nav: [
      { text: 'Getting Started', link: '/getting-started' },
      { text: 'Configuration', link: '/configuration' },
      { text: 'Contributing', link: '/contributing' },
      { text: 'GitHub', link: 'https://github.com/harrisonablack/minecart' },
    ],
    sidebar: [
      {
        items: [
          { text: 'Introduction', link: '/' },
          { text: 'Getting Started', link: '/getting-started' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Configuration', link: '/configuration' },
          {
            text: 'Server Types',
            items: [
              { text: 'Vanilla', link: '/server-types/vanilla' },
              { text: 'Paper', link: '/server-types/paper' },
            ],
          },
          { text: 'Plugins & Resources', link: '/plugins' },
        ],
      },
      {
        text: 'Development',
        items: [
          { text: 'Contributing', link: '/contributing' },
        ],
      },
    ],
    socialLinks: [
      { icon: 'github', link: 'https://github.com/harrisonablack/minecart' },
    ],
  },
})
