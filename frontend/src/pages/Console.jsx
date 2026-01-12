import { useEffect, useState } from "react";
import CreateEventCard from "../components/CreateEventCard.jsx";
import NotificationsTable from "../components/NotificationTable.jsx";

// This is a mock backend so the UI works immediately.
// Later you will replace this with real fetch calls to your Go API.
import { api } from "../api/taskcreater.js";

export default function Console() {
  // This state is shared by multiple UI sections, so it lives in the page.
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState("");

  // Polling interval: dashboards often poll because it's simple + reliable.
  const POLL_MS = 2000;

  async function refresh() {
    try {
      setErrorMsg("");
      const data = await api.listNotifications();
      console.log("data.items is ****",data.items)
      setItems(data.items);
      setLoading(false);
    } catch (err) {
      setErrorMsg(err.message || "Failed to load notifications");
      setLoading(false);
    }
  }
useEffect(() => {
  console.log("STATE items changed:", items, "len:", items?.length);
}, [items]);
  useEffect(() => {
    // Load once immediately, then poll.
    refresh();

    const id = setInterval(refresh, POLL_MS);
    return () => clearInterval(id); // cleanup prevents leaks on unmount
  }, []);

  return (
    <div className="page">
      <header className="header">
        <div>
          <h1 className="title">Safe-Notify</h1>
          <p className="subtitle">A Queue-based asynchronous notification pipeline</p>
        </div>

       
      </header>

      {/* Two-column layout: left card is wider than right card */}
      <section className="topGrid">
        <CreateEventCard
          onCreated={() => {
            // When an event is created, refresh immediately (don’t wait for next poll)
            refresh();
          }}
        />

        <div className="card">
          <h2 className="cardTitle">System Info</h2>
          <div className="kv">
            <div className="kvRow">
              <span className="kvKey">MAX_ATTEMPTS</span>
              <span className="kvVal">3</span>
            </div>
            <div className="kvRow">
              <span className="kvKey">Retry Backoff</span>
              <span className="kvVal">2s → 5s → DLQ</span>
            </div>
            <div className="kvRow">
              <span className="kvKey">Chaos Fail %</span>
              <span className="kvVal">Entered per ticket</span>
            </div>
          </div>

          <div className="legend">
            <div className="legendTitle">Legend</div>
            <div className="legendRow">
              <span className="pill pending">PENDING</span>
              <span className="pill failed">FAILED</span>
              <span className="pill dlq">DLQ</span>
              <span className="pill sent">SENT</span>
            </div>
          </div>
        </div>
      </section>

      <section className="card">
        <div className="tableHeader">
          <h2 className="cardTitle">Notifications</h2>
          <div className="tableMeta">
            {loading ? "Loading..." : `${items.length} items`}
          </div>
        </div>

        {errorMsg ? <div className="errorBanner">⚠ {errorMsg}</div> : null}

        <NotificationsTable
          items={items}
          onReplay={async (taskID) => {
    await api.replay({ task_id: taskID });
    refresh();
  }}
        />
      </section>
    </div>
  );
}

