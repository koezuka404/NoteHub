type HostBadgeProps = {
  role: string;
};

export default function HostBadge({ role }: HostBadgeProps) {
  if (role === 'host') {
    return <span className="host-badge">オーナー</span>;
  }
  if (role === 'member') {
    return <span className="role-badge">メンバー</span>;
  }
  return null;
}
