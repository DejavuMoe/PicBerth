// Ask the library screen to show an image's detail: used by the upload panel and the command palette,
// which can be used from any screen and any folder.

import { navigate, libraryPath } from './router.svelte';
import { tree } from './tree.svelte';

/** An image the library screen should open once its folder is on screen. */
export const openRequest = $state<{ id: string | null }>({ id: null });

export async function openImage(id: string, folderId: string | null): Promise<boolean> {
  const ok = await navigate(libraryPath(tree.names(folderId)));
  if (ok) openRequest.id = id;
  return ok;
}
