<script lang="ts">
  import type { ClientMessage, Color, GameStatus } from '../protocol/messages'

  interface Props {
    status: GameStatus
    turn: Color
    myColor: Color
    rolledForFirst: boolean
    disabled?: boolean
    onAction: (msg: ClientMessage) => void
  }

  let { status, turn, myColor, rolledForFirst, disabled = false, onAction }: Props = $props()

  // Игрок нажал «Бросить за первый ход» и ждёт соперника. Сервер молчит, пока
  // оба не пришлют ROLL_FOR_FIRST (FIRST_ROLL приходит только после второго),
  // так что подтверждения извне нет — помним факт клика сами. В рамках одной
  // партии rolledForFirst монотонно false→true, а новая партия размонтирует
  // ActionBar (App: client→null), поэтому сбрасывать флаг не нужно.
  let sentRollForFirst = $state(false)

  const myTurn = $derived(turn === myColor)
  const inFirstRollPhase = $derived(!rolledForFirst && status !== 'finished')
  const showRollForFirst = $derived(inFirstRollPhase && !sentRollForFirst)
  const showWaitingFirstRoll = $derived(inFirstRollPhase && sentRollForFirst)
  const showRoll = $derived(rolledForFirst && status === 'waitingForRoll' && myTurn)
  const showEndTurn = $derived(status === 'waitingForMove' && myTurn)
  const showResign = $derived(status !== 'finished')

  // При disabled (реконнект) кнопки инертны даже если клик дошёл до
  // делегированного обработчика — send по закрытому сокету бы бросил.
  function act(msg: ClientMessage): void {
    if (disabled) return
    if (msg.type === 'ROLL_FOR_FIRST') sentRollForFirst = true
    onAction(msg)
  }
</script>

<div class="action-bar">
  {#if showRollForFirst}
    <button
      data-testid="action-roll-for-first"
      type="button"
      {disabled}
      onclick={() => act({ type: 'ROLL_FOR_FIRST' })}
    >
      Бросить за первый ход
    </button>
  {/if}
  {#if showWaitingFirstRoll}
    <span class="waiting" role="status" aria-live="polite" data-testid="waiting-first-roll">
      <span class="spinner" aria-hidden="true"></span>
      Ожидание другого игрока…
    </span>
  {/if}
  {#if showRoll}
    <button
      data-testid="action-roll"
      type="button"
      {disabled}
      onclick={() => act({ type: 'ROLL' })}
    >
      Бросить кубики
    </button>
  {/if}
  {#if showEndTurn}
    <button
      data-testid="action-end-turn"
      type="button"
      {disabled}
      onclick={() => act({ type: 'END_TURN' })}
    >
      Завершить ход
    </button>
  {/if}
  {#if showResign}
    <button
      data-testid="action-resign"
      type="button"
      class="danger"
      {disabled}
      onclick={() => act({ type: 'RESIGN' })}
    >
      Сдаться
    </button>
  {/if}
</div>

<style>
  .action-bar {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  button {
    background: #f4ece1;
    color: #2a1e10;
    border: 2px solid #2a1e10;
    border-radius: 6px;
    padding: 0.5rem 0.875rem;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  button:hover {
    background: #e7c79b;
  }
  button.danger {
    background: #f4ece1;
    border-color: #8b1c1c;
    color: #8b1c1c;
  }
  button.danger:hover {
    background: #e8c4c4;
  }
  .waiting {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: #5a3a1e;
    font-size: 14px;
    font-weight: 600;
    font-style: italic;
  }
  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid #c19a6b;
    border-top-color: #5a3a1e;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .spinner {
      animation: none;
    }
  }
</style>
