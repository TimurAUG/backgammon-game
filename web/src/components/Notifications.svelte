<script lang="ts">
  // Контейнер тостов: рендерит стек из стора notifications, закрытие тоста
  // убирает его из стора. Фиксированный оверлей в углу — поверх игры.
  import { notifications, dismissNotification } from '../stores/notifications.svelte'

  import Toast from './Toast.svelte'
</script>

<div class="toasts" data-testid="notifications">
  {#each notifications.items as item (item.id)}
    <Toast text={item.text} onClose={() => dismissNotification(item.id)} />
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    pointer-events: none;
  }
  /* Сам контейнер прозрачен для кликов, тосты — кликабельны (крестик). */
  .toasts > :global(.toast) {
    pointer-events: auto;
  }
</style>
