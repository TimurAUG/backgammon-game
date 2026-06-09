<script lang="ts">
  import type { ClientMessage, Color, GameStatus } from '../protocol/messages'

  interface Props {
    status: GameStatus
    turn: Color
    myColor: Color
    rolledForFirst: boolean
    onAction: (msg: ClientMessage) => void
  }

  let { status, turn, myColor, rolledForFirst, onAction }: Props = $props()

  const myTurn = $derived(turn === myColor)
  const showRollForFirst = $derived(!rolledForFirst && status !== 'finished')
  const showRoll = $derived(rolledForFirst && status === 'waitingForRoll' && myTurn)
  const showEndTurn = $derived(status === 'waitingForMove' && myTurn)
  const showResign = $derived(status !== 'finished')
</script>

<div class="action-bar">
  {#if showRollForFirst}
    <button
      data-testid="action-roll-for-first"
      type="button"
      onclick={() => onAction({ type: 'ROLL_FOR_FIRST' })}
    >
      Бросить за первый ход
    </button>
  {/if}
  {#if showRoll}
    <button
      data-testid="action-roll"
      type="button"
      onclick={() => onAction({ type: 'ROLL' })}
    >
      Бросить кубики
    </button>
  {/if}
  {#if showEndTurn}
    <button
      data-testid="action-end-turn"
      type="button"
      onclick={() => onAction({ type: 'END_TURN' })}
    >
      Завершить ход
    </button>
  {/if}
  {#if showResign}
    <button
      data-testid="action-resign"
      type="button"
      class="danger"
      onclick={() => onAction({ type: 'RESIGN' })}
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
</style>
