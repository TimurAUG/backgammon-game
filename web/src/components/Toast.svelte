<script lang="ts">
  // Одна плашка-уведомление. role="status" + aria-live="polite" — скринридер
  // озвучит появление, не прерывая текущую речь. Закрывается крестиком или
  // сама по таймеру ttl.
  interface Props {
    text: string
    onClose: () => void
    ttl?: number
  }

  let { text, onClose, ttl = 5000 }: Props = $props()

  // $effect (не $derived) — авто-скрытие это императивный таймер с очисткой,
  // декларативно не выражается. Очистка снимает таймер при размонтировании
  // (например, когда тост закрыли крестиком раньше срока).
  $effect(() => {
    const timer = setTimeout(onClose, ttl)
    return () => clearTimeout(timer)
  })
</script>

<div class="toast" role="status" aria-live="polite">
  <span class="toast-text">{text}</span>
  <button type="button" class="toast-close" aria-label="Закрыть" onclick={onClose}>×</button>
</div>

<style>
  .toast {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    background: #2a1e10;
    color: #f4ece1;
    border: 1px solid #c19a6b;
    border-radius: 8px;
    padding: 0.6rem 0.75rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
    font-size: 14px;
    min-width: 220px;
  }
  .toast-text {
    flex: 1;
  }
  .toast-close {
    background: transparent;
    border: none;
    color: #e7c79b;
    font-size: 18px;
    line-height: 1;
    cursor: pointer;
    padding: 0 0.2rem;
  }
  .toast-close:hover {
    color: #fff;
  }
</style>
