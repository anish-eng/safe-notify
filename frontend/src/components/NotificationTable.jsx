// import StatusPill from "./Statuspill.jsx";
function fmtTime(ms) {
  if (!ms) return "—";
  const n = Number(ms);
  if (!Number.isFinite(n)) return String(ms);
  return new Date(n).toLocaleTimeString();
}
export default function NotificationsTable({ items, onReplay }) {
  
  return (
    <div className="tableWrap">
      <table className="table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Entity ID</th>
            <th>Channel</th>
            <th>Status</th>
            <th>Attempts</th>
            <th>Last Error</th>
            <th>Action</th>
          </tr>
        </thead>

        <tbody>
          {items.length === 0 ? (
            <tr>
              <td colSpan="7" className="empty">
                No notifications yet. Create a ticket event above.
              </td>
            </tr>
          ) : (
            items.map((it) => (
              <tr key={it.idempotency_key || it.task_id || `${it.entity_id}-${it.created_at}`}>
                <td className="mono">{fmtTime(it.updated_at || it.created_at || "—")}</td>
                <td className="mono">{it.entity_id || "—"}</td>
                <td>{it.channel || "EMAIL"}</td>
                <td>
                  {it.status === "PENDING" ? (
    <div className="legendRow">
      <span className="pill pending">PENDING</span>
    </div>
  ) : it.status === "FAILED" ? (
    <div className="legendRow">
      <span className="pill failed">FAILED</span>
    </div>
  ) : it.status === "DLQ" ? (
    <div className="legendRow">
      <span className="pill dlq">DLQ</span>
    </div>
  ) : it.status === "SENT" ? (
    <div className="legendRow">
      <span className="pill sent">SENT</span>
    </div>
  ) : null}
                  
                </td>
                <td className="mono">{String(it.attempt_count ?? 0)}</td>
                <td className="muted">{it.last_error || "—"}</td>
                <td>
                  {it.status === "DLQ" ? (
                    <button
                      className="btn small"
                       onClick={() => onReplay?.(it.task_id)}
                      disabled={!it.task_id}
                    >
                      Replay
                    </button>
                  ) : (
                    <span className="muted">—</span>
                  )}
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
