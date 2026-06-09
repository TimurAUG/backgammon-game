<script lang="ts">
  import type { Color, WinKind } from '../protocol/messages'

  interface Props {
    gameOver: { winner: Color; kind: WinKind } | null
    myColor?: Color | null
    onNewGame: () => void
  }

  let { gameOver, myColor = null, onNewGame }: Props = $props()

  function colorLabel(c: Color): string {
    return c === 'white' ? 'Белые' : 'Чёрные'
  }

  function kindLabel(k: WinKind): string {
    switch (k) {
      case 'oin':
        return 'Оин'
      case 'mars':
        return 'Марс'
      case 'koks':
        return 'Кокс'
    }
  }

  function outcomeLabel(winner: Color): string {
    if (myColor === null) return `${colorLabel(winner)} победили`
    return myColor === winner ? 'Вы победили' : 'Вы проиграли'
  }
</script>

{#if gameOver}
  <div class="overlay" data-testid="game-over" role="dialog" aria-modal="true">
    <div class="card">
      <h2 class="title">{outcomeLabel(gameOver.winner)}</h2>
      <p class="subtitle">
        {colorLabel(gameOver.winner)} · {kindLabel(gameOver.kind)}
      </p>
      <button
        type="button"
        class="primary"
        data-testid="action-new-game"
        onclick={onNewGame}
      >
        Новая игра
      </button>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  .card {
    background: #f4ece1;
    color: #2a1e10;
    border: 2px solid #2a1e10;
    border-radius: 12px;
    padding: 1.5rem 2rem;
    text-align: center;
    min-width: 260px;
  }
  .title {
    margin: 0 0 0.25rem;
    font-size: 1.4rem;
  }
  .subtitle {
    margin: 0 0 1rem;
    opacity: 0.8;
  }
  .primary {
    background: #2a1e10;
    color: #f4ece1;
    border: none;
    border-radius: 6px;
    padding: 0.6rem 1.1rem;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  .primary:hover {
    background: #5a3a1e;
  }
</style>
