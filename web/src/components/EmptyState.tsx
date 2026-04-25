import type { ReactNode } from "react";

type EmptyStateProps = {
  title: string;
  description: string;
  action?: ReactNode;
};

export default function EmptyState(props: EmptyStateProps) {
  return (
    <div className="empty-state">
      <div className="empty-state-mark" aria-hidden="true">
        TM
      </div>
      <div className="empty-state-copy">
        <strong>{props.title}</strong>
        <p>{props.description}</p>
      </div>
      {props.action ? <div className="empty-state-action">{props.action}</div> : null}
    </div>
  );
}
