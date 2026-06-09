<script lang="ts">
  import { saveCredentials, type Credentials } from '../lib/credentials'

  let { onConnect }: { onConnect: (creds: Credentials) => void } = $props()

  let gameId = $state('')
  let token = $state('')

  function handleSubmit(e: SubmitEvent): void {
    e.preventDefault()
    const creds = { gameId: gameId.trim(), token: token.trim() }
    if (creds.gameId === '' || creds.token === '') return
    saveCredentials(creds)
    onConnect(creds)
  }
</script>

<main class="connect">
  <h1>Длинные нарды</h1>
  <form onsubmit={handleSubmit}>
    <label>
      ID игры
      <input type="text" bind:value={gameId} autocomplete="off" />
    </label>
    <label>
      Токен
      <input type="password" bind:value={token} autocomplete="off" />
    </label>
    <button type="submit">Подключиться</button>
  </form>
</main>

<style>
  .connect {
    max-width: 320px;
    margin: 4rem auto;
    color: #2a1e10;
  }
  h1 {
    text-align: center;
    font-size: 1.5rem;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 14px;
    font-weight: 600;
  }
  input {
    border: 2px solid #2a1e10;
    border-radius: 6px;
    padding: 0.5rem;
    font-size: 14px;
  }
  button {
    margin-top: 0.5rem;
    background: #2a1e10;
    color: #f4ece1;
    border: none;
    border-radius: 6px;
    padding: 0.6rem;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }
  button:hover {
    background: #5a3a1e;
  }
</style>
