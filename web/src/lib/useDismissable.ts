import { useEffect, useRef } from "react";

// 让下拉菜单/浮层在点击外部或按 Esc 时关闭。返回的 ref 挂到浮层的包裹元素上。
export function useDismissable<T extends HTMLElement>(open: boolean, onDismiss: () => void) {
  const ref = useRef<T | null>(null);
  const onDismissRef = useRef(onDismiss);
  onDismissRef.current = onDismiss;

  useEffect(() => {
    if (!open) {
      return;
    }

    function handlePointerDown(event: PointerEvent) {
      if (event.target instanceof Node && ref.current?.contains(event.target)) {
        return;
      }

      onDismissRef.current();
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onDismissRef.current();
      }
    }

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  return ref;
}
