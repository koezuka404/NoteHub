import type { ReactNode } from 'react';

type HostPanelProps = {
  title: string;
  description?: string;
  children: ReactNode;
};

export default function HostPanel({ title, description, children }: HostPanelProps) {
  return (
    <section className="host-panel" aria-label={title}>
      <h3 className="host-panel-title">{title}</h3>
      {description ? <p className="host-panel-description">{description}</p> : null}
      {children}
    </section>
  );
}
