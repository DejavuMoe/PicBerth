// Touch long-press: runs `fn` after a held touch that did not move.

export function longpress(node: HTMLElement, fn: () => void) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  let x = 0;
  let y = 0;
  const cancel = () => clearTimeout(timer);
  const down = (e: PointerEvent) => {
    if (e.pointerType !== 'touch') return;
    x = e.clientX;
    y = e.clientY;
    timer = setTimeout(() => fn(), 450);
  };
  const move = (e: PointerEvent) => {
    if (Math.abs(e.clientX - x) > 8 || Math.abs(e.clientY - y) > 8) cancel();
  };
  node.addEventListener('pointerdown', down);
  node.addEventListener('pointermove', move);
  node.addEventListener('pointerup', cancel);
  node.addEventListener('pointercancel', cancel);
  node.addEventListener('pointerleave', cancel);
  return {
    update(next: () => void) {
      fn = next;
    },
    destroy() {
      cancel();
      node.removeEventListener('pointerdown', down);
      node.removeEventListener('pointermove', move);
      node.removeEventListener('pointerup', cancel);
      node.removeEventListener('pointercancel', cancel);
      node.removeEventListener('pointerleave', cancel);
    },
  };
}
