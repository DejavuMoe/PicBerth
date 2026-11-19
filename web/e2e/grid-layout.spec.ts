import { test, expect } from '@playwright/test';
import { createGridLayout } from '../src/components/library/grid-layout';
import type { Image } from '../src/lib/types';

test('grid keeps finished rows on append and recomputes from the first change', () => {
  const layout = createGridLayout();
  const images = Array.from({ length: 12 }, (_, id) => ({ id: String(id), width: 1, height: 1 } as Image));
  const before = layout(images.slice(0, 7), 600);
  expect(before.map(row => row.items.length)).toEqual([4, 3]);
  expect(before.map(row => row.h)).toEqual([141, 188]);
  const appended = layout(images, 600);
  expect(appended[0]).toBe(before[0]);
  expect(appended).toEqual(createGridLayout()(images, 600));
  const removed = images.splice(2, 1);
  expect(layout(images, 600)).toEqual(createGridLayout()(images, 600));
  images.splice(2, 0, ...removed);
  expect(layout(images, 600)).toEqual(createGridLayout()(images, 600));
  images[1]!.width = 2;
  expect(layout(images, 600)).toEqual(createGridLayout()(images, 600));
  images[1] = { ...images[1]!, filename: 'renamed.png' };
  expect(layout(images, 600)[0]!.items[1]!.im).toBe(images[1]);
  expect(layout(images, 390)).toEqual(createGridLayout()(images, 390));
  expect(layout([], 390)).toEqual([]);
});
