import type { Image } from '../../lib/types';

interface Row {
  h: number;
  items: { im: Image; r: number }[];
}

function ratio(im: Image) {
  return Math.max(0.6, Math.min(2.4, im.width / im.height || 1));
}

export function createGridLayout() {
  let previous: Row[] = [];
  let previousWidth = 0;

  return (list: readonly Image[], width: number): Row[] => {
    const narrow = width < 560;
    const target = narrow ? 132 : 188;
    const gap = narrow ? 8 : 12;
    const rows: Row[] = [];
    let offset = 0;

    // The last row may still accept tiles. Preserve complete rows until the first changed tile.
    if (width === previousWidth) {
      for (let i = 0; i < previous.length - 1; i++) {
        const row = previous[i]!;
        if (!row.items.every(({ im, r }, j) => list[offset + j] === im && ratio(im) === r)) break;
        rows.push(row);
        offset += row.items.length;
      }
    }

    let items: Row['items'] = [];
    let sum = 0;
    for (; offset < list.length; offset++) {
      const im = list[offset]!;
      const r = ratio(im);
      items.push({ im, r });
      sum += r;
      const h = (width - gap * (items.length - 1)) / sum;
      if (h <= target) {
        rows.push({ h, items });
        items = [];
        sum = 0;
      }
    }
    if (items.length) rows.push({ h: Math.min(target, (width - gap * (items.length - 1)) / sum), items });
    previous = rows;
    previousWidth = width;
    return rows;
  };
}
