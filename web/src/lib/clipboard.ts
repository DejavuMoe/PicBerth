export type CopyResult = 'copied' | 'selected' | 'failed';

export const INSECURE_HINT = '当前连接不安全，已选中链接，请按 Ctrl / ⌘ + C';

/**
 * Copy text. The async clipboard API needs a secure context; otherwise, when a text control is given,
 * its text is selected so the user can press Ctrl/⌘+C. Without a control a hidden textarea and the
 * legacy command are tried as a last resort.
 */
export async function copyText(
  text: string,
  field?: HTMLInputElement | HTMLTextAreaElement | null,
): Promise<CopyResult> {
  if (window.isSecureContext && navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      return 'copied';
    } catch {
      // permission denied or document not focused: fall through
    }
  }
  if (field) {
    field.focus();
    field.select();
    return 'selected';
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.setAttribute('readonly', '');
  ta.style.position = 'fixed';
  ta.style.top = '0';
  ta.style.opacity = '0';
  document.body.append(ta);
  ta.select();
  let ok = false;
  try {
    ok = document.execCommand('copy');
  } catch {
    ok = false;
  }
  ta.remove();
  return ok ? 'copied' : 'failed';
}
