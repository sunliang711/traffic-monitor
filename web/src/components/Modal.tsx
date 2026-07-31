import { useEffect, useRef } from "react";
import type { MouseEvent, ReactNode } from "react";
import { useI18n } from "../lib/i18n";

type ModalProps = {
  title: ReactNode;
  onClose: () => void;
  children: ReactNode;
  className?: string;
};

export default function Modal(props: ModalProps) {
  const { t } = useI18n();
  const onCloseRef = useRef(props.onClose);
  onCloseRef.current = props.onClose;

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onCloseRef.current();
      }
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = previousOverflow;
    };
  }, []);

  function handleBackdropMouseDown(event: MouseEvent<HTMLDivElement>) {
    // 仅点击遮罩空白处时关闭，避免面板内选中文本拖拽误触。
    if (event.target === event.currentTarget) {
      props.onClose();
    }
  }

  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={handleBackdropMouseDown}>
      <section className={`modal-panel${props.className ? ` ${props.className}` : ""}`} aria-modal="true" role="dialog">
        <div className="modal-header">
          <div>
            <h3 className="panel-title">{props.title}</h3>
          </div>
          <button className="secondary-button modal-close-button" onClick={props.onClose} type="button">
            {t("close")}
          </button>
        </div>
        {props.children}
      </section>
    </div>
  );
}
