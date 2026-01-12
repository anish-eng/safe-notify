import { useEffect, useState } from "react";
import { api } from "../api/taskcreater.js";

function generateEntityId() {
  // Simple readable IDs for demo: TICKET-AB3F
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  let code = "";
  for (let i = 0; i < 4; i++) {
    code += chars[Math.floor(Math.random() * chars.length)];
}
  return `TICKET-${code}`;
}

export default function CreateEventCard({ onCreated }) {
  // Form state lives here because only this component uses it.
  const [entityId, setEntityId] = useState("");
  const [recipientEmail, setRecipientEmail] = useState("demo@example.com");
  const [priority, setPriority] = useState("HIGH");
  const [chaosFailPercent, setChaosFailPercent] = useState("0");

  const [submitting, setSubmitting] = useState(false);
  const [statusMsg, setStatusMsg] = useState("");

  useEffect(() => {
    // Auto-fill the entity ID on first load (better UX).
    setEntityId(generateEntityId());
  }, []);

  function validate() {
    const n = Number(chaosFailPercent);
    if (!entityId.trim()) return "Entity ID is required";
    if (!recipientEmail.trim()) return "Recipient email is required";
    if (Number.isNaN(n) || n < 0 || n > 100) return "Chaos Fail % must be 0–100";
    return "";
  }

  async function SubmitHandler(e) {
    e.preventDefault(); // prevents page reload
    setStatusMsg("");

    const v = validate();
    if (v) {
      setStatusMsg(`❌ ${v}`);
      return;
    }

    setSubmitting(true);
    try {
      // This payload matches what your Go /events endpoint will later accept.
      const payload = {
        eventType: "ticket_escalated",
        entityId: entityId.trim(),
        priority:priority,
        recipientEmail: recipientEmail.trim(),
        chaosFailPercent: Number(chaosFailPercent),
      };

      const res = await api.createEvent(payload);
      console.log("res from events backend",res)
      setStatusMsg(`✅ Notification created(taskId: ${res.task_id}),EntityId:${res.entity_id}`);

      // After a successful send, generate a new ID for the next ticket.
      setEntityId(generateEntityId());

      // Tell the parent page to refresh.
      onCreated?.();
    } catch (err) {
      setStatusMsg(`❌ ${err.message || "Failed to create event"}`);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="card">
      <h2 className="cardTitle">Create Ticket Event</h2>

      <form className="form" onSubmit={SubmitHandler}>
        <div className="row">
          <label className="label">Event Type</label>
          <div className="staticValue">ticket_escalated</div>
        </div>

        <div className="row">
          <label className="label">Entity ID</label>
          <div className="inline">
            <input
              className="input"
              value={entityId}
              onChange={(e) => setEntityId(e.target.value)}
              placeholder="TICKET-AB3F"
            />
            <button
              type="button"
              className="btn secondary"
              onClick={() => setEntityId(generateEntityId())}
            >
              Generate
            </button>
          </div>
        </div>

        <div className="row">
          <label className="label">Recipient Email</label>
          <input
            className="input"
            value={recipientEmail}
            onChange={(e) => setRecipientEmail(e.target.value)}
            placeholder="demo@example.com"
          />
        </div>

        <div className="row twoCol">
          <div>
            <label className="label">Priority</label>
            <select className="input" value={priority} onChange={(e) => setPriority(e.target.value)}>
              <option value="LOW">LOW</option>
              <option value="MED">MED</option>
              <option value="HIGH">HIGH</option>
            </select>
          </div>

          <div>
            <label className="label">Chaos Fail %</label>
            <input
              className="input"
              value={chaosFailPercent}
              onChange={(e) => setChaosFailPercent(e.target.value)}
              placeholder="0"
            />
          </div>
        </div>

        <button className="btn primary" type="submit" disabled={submitting}>
          {submitting ? "Sending..." : "Send Event"}
        </button>

        {statusMsg ? <div className="statusMsg">{statusMsg}</div> : null}
      </form>
    </div>
  );
}
