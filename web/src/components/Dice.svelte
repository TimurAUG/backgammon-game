<script lang="ts">
  import type { Dice } from '../protocol/messages'

  let { dice }: { dice: Dice | null } = $props()
</script>

{#if dice}
  <div class="dice">
    <div class="pair">
      <div class="die" data-testid="die-a">{dice.a}</div>
      <div class="die" data-testid="die-b">{dice.b}</div>
    </div>
    {#if dice.remaining.length > 0}
      <ul class="remaining" aria-label="Оставшиеся пипсы">
        {#each dice.remaining as pip, i (i)}
          <li class="pip" data-testid="remaining-{i}">{pip}</li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}

<style>
  .dice {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    align-items: center;
  }
  .pair {
    display: flex;
    gap: 0.5rem;
  }
  .die {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f4ece1;
    color: #2a1e10;
    border: 2px solid #2a1e10;
    border-radius: 8px;
    font-size: 24px;
    font-weight: 700;
  }
  .remaining {
    display: flex;
    gap: 0.25rem;
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .pip {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fff;
    border: 1px solid #2a1e10;
    border-radius: 50%;
    font-size: 14px;
    font-weight: 600;
  }
</style>
