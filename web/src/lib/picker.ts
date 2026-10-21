// One hidden file input owned by the shell; any button can open it.

let input: HTMLInputElement | null = null;

export function registerPicker(el: HTMLInputElement | null) {
  input = el;
}

export function openPicker() {
  input?.click();
}
