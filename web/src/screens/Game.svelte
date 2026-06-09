<script lang="ts">
  import ActionBar from '../components/ActionBar.svelte'
  import Board from '../components/Board.svelte'
  import Dice from '../components/Dice.svelte'
  import GameOver from '../components/GameOver.svelte'
  import type { ClientMessage } from '../protocol/messages'
  import { connection } from '../stores/connection.svelte'
  import { gameState } from '../stores/game.svelte'

  interface Props {
    onAction: (msg: ClientMessage) => void
    onNewGame: () => void
  }

  let { onAction, onNewGame }: Props = $props()

  // Первый бросок уже состоялся, если кубики на столе или кто-то
  // уже завершил свой первый ход (после END_TURN кубики сбрасываются).
  const rolledForFirst = $derived(
    gameState.dice !== null || !gameState.isFirstMove.white || !gameState.isFirstMove.black,
  )

  // Реконнект → ActionBar заблокирован (сокет закрыт, send бы бросил).
  const blocked = $derived(connection.state === 'reconnecting')

  function handleMove(from: number, to: number): void {
    onAction({ type: 'MOVE', from, to })
  }
</script>

<main class="game">
  <Board
    board={gameState.board}
    legalMoves={gameState.legalMoves}
    myColor={gameState.myColor}
    onMove={handleMove}
  />
  <aside class="side">
    <Dice dice={gameState.dice} />
    {#if gameState.myColor !== null}
      <ActionBar
        status={gameState.status}
        turn={gameState.turn}
        myColor={gameState.myColor}
        {rolledForFirst}
        disabled={blocked}
        {onAction}
      />
    {/if}
  </aside>
  <GameOver gameOver={gameState.gameOver} myColor={gameState.myColor} {onNewGame} />
</main>

<style>
  .game {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 800px;
    margin: 1rem auto;
    padding: 0 1rem;
  }
  .side {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }
</style>
