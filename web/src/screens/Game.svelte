<script lang="ts">
  import ActionBar from '../components/ActionBar.svelte'
  import BearOffCounter from '../components/BearOffCounter.svelte'
  import Board from '../components/Board.svelte'
  import Dice from '../components/Dice.svelte'
  import GameOver from '../components/GameOver.svelte'
  import Chat from '../components/Chat.svelte'
  import type { ClientMessage } from '../protocol/messages'
  import { playRollCue } from '../lib/sound'
  import { connection } from '../stores/connection.svelte'
  import { gameState } from '../stores/game.svelte'
  import { pushNotification } from '../stores/notifications.svelte'

  interface Props {
    onAction: (msg: ClientMessage) => void
    onNewGame: () => void
    gameId?: string | null
    token?: string | null
  }

  let { onAction, onNewGame, gameId = null, token = null }: Props = $props()

  // Первый бросок уже состоялся, если был FIRST_ROLL, или кубики на столе,
  // или кто-то уже завершил свой первый ход. После розыгрыша победитель
  // бросает заново (dice=null до его ROLL) — опираться только на dice нельзя.
  const rolledForFirst = $derived(
    gameState.firstRoll !== null ||
      gameState.dice !== null ||
      !gameState.isFirstMove.white ||
      !gameState.isFirstMove.black,
  )

  // Реконнект → ActionBar заблокирован (сокет закрыт, send бы бросил).
  const blocked = $derived(connection.state === 'reconnecting')

  // Первый бросок: кто сколько бросил (#2).
  const firstRoll = $derived(gameState.firstRoll)

  // Плашку первого броска показываем только в окне «розыгрыш прошёл, но
  // победитель ещё не бросил кубики на ход»: waitingForRoll и оба игрока
  // ещё не начинали (isFirstMove). После ROLL победителя (waitingForMove) и
  // после первого END_TURN — скрываем и больше не возвращаем.
  const showFirstRollBanner = $derived(
    firstRoll !== null &&
      gameState.status === 'waitingForRoll' &&
      gameState.isFirstMove.white &&
      gameState.isFirstMove.black,
  )

  // «Ожидается мой бросок»: на обычном ходу — только в свою очередь, на стадии
  // розыгрыша первого хода (rolledForFirst=false) — всегда, т.к. бросают оба.
  // started отсекает initial-снапшот между JOINED и первым STATE (#34b).
  const awaitingMyRoll = $derived(
    gameState.started &&
      gameState.myColor !== null &&
      gameState.status === 'waitingForRoll' &&
      (rolledForFirst ? gameState.turn === gameState.myColor : true),
  )

  // Детектор перехода false→true: показываем тост «Твой бросок» и проигрываем
  // сигнал. $effect (не $derived) — потому что это императивная побочка (звук)
  // плюс сравнение с предыдущим значением. Первый прогон лишь фиксирует базу,
  // чтобы не звякнуть на самом маунте компонента.
  let prevAwaitingRoll = false
  let rollCueReady = false
  $effect(() => {
    const awaiting = awaitingMyRoll
    if (!rollCueReady) {
      rollCueReady = true
      prevAwaitingRoll = awaiting
      return
    }
    if (awaiting && !prevAwaitingRoll) {
      pushNotification('yourRoll', 'Твой бросок')
      playRollCue()
    }
    prevAwaitingRoll = awaiting
  })

  function handleMove(from: number, to: number): void {
    onAction({ type: 'MOVE', from, to })
  }

  // Личная ссылка для возврата в эту игру (в т.ч. с другого устройства):
  // содержит token, поэтому она НЕ для соперника — только для себя.
  const reconnectLink = $derived(
    gameId !== null && token !== null
      ? `${location.origin}/?game=${gameId}&token=${token}`
      : null,
  )

  function copyReconnect(): void {
    if (reconnectLink !== null) void navigator.clipboard?.writeText(reconnectLink)
  }
</script>

<main class="game">
  <header class="toolbar">
    {#if reconnectLink !== null}
      <div class="reconnect">
        <span class="reconnect-label">Ссылка для возврата:</span>
        <input
          class="reconnect-link"
          readonly
          value={reconnectLink}
          data-testid="reconnect-link"
          title="Открой эту ссылку на другом устройстве, чтобы вернуться в игру"
        />
        <button type="button" data-testid="copy-reconnect" onclick={copyReconnect}>
          Копировать
        </button>
      </div>
    {/if}
    <button type="button" class="leave" data-testid="switch-game" onclick={onNewGame}>
      Сменить игру
    </button>
  </header>

  {#if blocked}
    <div class="reconnecting" role="status" aria-live="polite" data-testid="reconnecting-banner">
      Переподключение…
    </div>
  {/if}

  {#if showFirstRollBanner && firstRoll}
    <div class="first-roll" data-testid="first-roll-banner">
      Первый бросок — Белые: <b>{firstRoll.white}</b>, Чёрные: <b>{firstRoll.black}</b>. Первым ходит
      {firstRoll.white > firstRoll.black ? 'Белые' : 'Чёрные'}.
    </div>
  {/if}

  <Board
    board={gameState.board}
    legalMoves={gameState.legalMoves}
    reach={gameState.reach}
    myColor={gameState.myColor}
    onMove={handleMove}
  />
  <aside class="side">
    <BearOffCounter borneOff={gameState.borneOff} allHome={gameState.allHome} />
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
  <Chat myColor={gameState.myColor} {onAction} />
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
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    justify-content: flex-end;
  }
  .reconnect {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex: 1;
    min-width: 0;
  }
  .reconnect-label {
    font-size: 12px;
    color: #5a3a1e;
    white-space: nowrap;
  }
  .reconnect-link {
    flex: 1;
    min-width: 0;
    border: 1px solid #c19a6b;
    border-radius: 4px;
    padding: 0.3rem 0.4rem;
    font-size: 12px;
    color: #2a1e10;
    background: #f4ece1;
  }
  .reconnect button {
    border: 1px solid #5a3a1e;
    background: #f4ece1;
    border-radius: 4px;
    padding: 0.3rem 0.6rem;
    font-size: 12px;
    font-weight: 600;
    color: #2a1e10;
    cursor: pointer;
    white-space: nowrap;
  }
  .leave {
    background: transparent;
    color: #5a3a1e;
    border: 1px solid #5a3a1e;
    border-radius: 6px;
    padding: 0.35rem 0.75rem;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }
  .leave:hover {
    background: #e7c79b;
  }
  .first-roll {
    background: #f4ece1;
    border: 1px solid #c19a6b;
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
    font-size: 14px;
    color: #2a1e10;
    text-align: center;
  }
  .reconnecting {
    background: #e7c79b;
    border: 1px solid #5a3a1e;
    border-radius: 6px;
    padding: 0.5rem 0.75rem;
    font-size: 14px;
    font-weight: 600;
    color: #2a1e10;
    text-align: center;
  }
</style>
