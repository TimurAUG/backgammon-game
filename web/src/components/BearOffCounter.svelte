<script lang="ts">
  import type { Color } from '../protocol/messages'

  interface Props {
    borneOff: { white: number; black: number }
    allHome: { white: boolean; black: boolean }
  }

  let { borneOff, allHome }: Props = $props()

  // Всего шашек у каждого игрока — фиксировано правилами длинных нард.
  // «Осталось сбросить» = TOTAL − выкинуто (borneOff): чистая арифметика от
  // серверных данных, без дублирования правил игры на клиенте.
  const TOTAL = 15

  const colors: Color[] = ['white', 'black']

  function colorLabel(c: Color): string {
    return c === 'white' ? 'Белые' : 'Чёрные'
  }
</script>

<!-- Плашка видна, только когда хотя бы у одного цвета все шашки в доме
     (allHome — домен AllInHome из STATE): счётчик осмыслен лишь в фазе сброса. -->
{#if allHome.white || allHome.black}
  <div class="bear-off-counter" data-testid="bear-off-counter">
    <span class="title">Осталось сбросить</span>
    {#each colors as c (c)}
      {#if allHome[c]}
        <div class="row" data-testid="bear-off-remaining-{c}">
          <span class="dot {c}"></span>
          <span class="who">{colorLabel(c)}</span>
          <span class="count">{TOTAL - borneOff[c]}/{TOTAL}</span>
        </div>
      {/if}
    {/each}
  </div>
{/if}

<style>
  .bear-off-counter {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    background: #f4ece1;
    border: 1px solid #c19a6b;
    border-radius: 6px;
    padding: 0.4rem 0.6rem;
    font-size: 13px;
    color: #2a1e10;
  }
  .title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: #5a3a1e;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 1px solid #2a1e10;
    flex: none;
  }
  .dot.white {
    background: #f4ece1;
  }
  .dot.black {
    background: #2a1e10;
  }
  .who {
    flex: 1;
  }
  .count {
    font-variant-numeric: tabular-nums;
    font-weight: 700;
  }
</style>
