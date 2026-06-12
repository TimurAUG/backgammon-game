import { mount } from 'svelte'
import App from './App.svelte'
import { setupTelegram } from './lib/telegram'

// Внутри Telegram-шита отключаем вертикальный свайп — иначе он перехватывает
// drag шашек (#drag-touch). Вне Telegram — no-op.
setupTelegram(window.Telegram?.WebApp)

const target = document.getElementById('app')
if (!target) {
  throw new Error('mount target #app not found')
}

const app = mount(App, { target })

export default app
