<script lang="ts">
  import type { ClientMessage, Color } from '../protocol/messages'
  import { chat, setChatOpen } from '../stores/chat.svelte'

  interface Props {
    myColor: Color | null
    onAction: (msg: ClientMessage) => void
  }

  let { myColor, onAction }: Props = $props()

  let draft = $state('')

  function colorLabel(c: Color): string {
    return c === 'white' ? 'Белые' : 'Чёрные'
  }

  // Шлём только непустой текст; сервер всё равно тримит и валидирует, но
  // пустое отправлять незачем. После отправки поле очищается.
  function send(): void {
    const text = draft.trim()
    if (text === '') return
    onAction({ type: 'CHAT', text })
    draft = ''
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      send()
    }
  }

  // Разворот = «прочитано» (setChatOpen(true) обнуляет unread в сторе).
  function open(): void {
    setChatOpen(true)
  }

  function close(): void {
    setChatOpen(false)
  }
</script>

{#if chat.open}
  <section class="chat" data-testid="chat-panel">
    <header class="chat-header">
      <span class="title">Чат</span>
      <button
        data-testid="chat-close"
        type="button"
        class="icon"
        onclick={close}
        aria-label="Свернуть чат"
      >
        ×
      </button>
    </header>
    <ul class="messages">
      {#each chat.messages as m, i (i)}
        <li class="msg" class:mine={m.sender === myColor} data-testid="chat-message">
          <span class="who">{colorLabel(m.sender)}</span>
          <span class="text">{m.text}</span>
        </li>
      {/each}
    </ul>
    <div class="compose">
      <input
        data-testid="chat-input"
        type="text"
        placeholder="Сообщение…"
        bind:value={draft}
        onkeydown={onKeydown}
      />
      <button data-testid="chat-send" type="button" onclick={send}>Отправить</button>
    </div>
  </section>
{:else}
  <button
    class="chat-toggle"
    data-testid="chat-toggle"
    type="button"
    onclick={open}
    aria-label="Открыть чат"
  >
    💬 Чат
    {#if chat.unread > 0}
      <span class="badge" data-testid="chat-unread">{chat.unread}</span>
    {/if}
  </button>
{/if}

<style>
  .chat {
    position: fixed;
    right: 1rem;
    bottom: 1rem;
    z-index: 50;
    width: min(340px, calc(100vw - 2rem));
    max-height: min(60vh, 480px);
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    border: 1px solid #c19a6b;
    border-radius: 8px;
    background: #f4ece1;
    padding: 0.5rem;
    box-shadow: 0 6px 24px rgba(42, 30, 16, 0.25);
  }
  .messages {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    overflow-y: auto;
    flex: 1;
    min-height: 0;
  }
  .msg {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    max-width: 85%;
    padding: 0.3rem 0.5rem;
    border-radius: 8px;
    background: #fff7ec;
    border: 1px solid #d9c3a0;
  }
  .msg.mine {
    align-self: flex-end;
    align-items: flex-end;
    background: #e7c79b;
  }
  .who {
    font-size: 11px;
    font-weight: 600;
    color: #5a3a1e;
  }
  .text {
    font-size: 14px;
    color: #2a1e10;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .compose {
    display: flex;
    gap: 0.4rem;
  }
  .compose input {
    flex: 1;
    min-width: 0;
    border: 1px solid #c19a6b;
    border-radius: 6px;
    padding: 0.4rem 0.5rem;
    font-size: 14px;
    color: #2a1e10;
    background: #fff;
  }
  .compose button {
    border: 1px solid #2a1e10;
    background: #f4ece1;
    border-radius: 6px;
    padding: 0.4rem 0.75rem;
    font-size: 13px;
    font-weight: 600;
    color: #2a1e10;
    cursor: pointer;
  }
  .compose button:hover {
    background: #e7c79b;
  }
  .chat-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .title {
    font-weight: 700;
    font-size: 14px;
    color: #2a1e10;
  }
  .icon {
    border: none;
    background: transparent;
    font-size: 18px;
    line-height: 1;
    color: #5a3a1e;
    cursor: pointer;
    padding: 0 0.25rem;
  }
  .chat-toggle {
    position: fixed;
    right: 1rem;
    bottom: 1rem;
    z-index: 50;
    border: 1px solid #2a1e10;
    background: #f4ece1;
    border-radius: 999px;
    padding: 0.5rem 0.9rem;
    font-size: 14px;
    font-weight: 600;
    color: #2a1e10;
    cursor: pointer;
  }
  .chat-toggle:hover {
    background: #e7c79b;
  }
  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    margin-left: 0.35rem;
    padding: 0 5px;
    border-radius: 999px;
    background: #8b1c1c;
    color: #fff;
    font-size: 11px;
    font-weight: 700;
  }
</style>
