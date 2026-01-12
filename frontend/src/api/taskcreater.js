

// function randCode(n = 4) {
//   const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
//   let s = "";
//   for (let i = 0; i < n; i++) s += chars[Math.floor(Math.random() * chars.length)];
//   return s;
// }

// function makeEntityId() {
//   return `TICKET-${randCode(4)}`;
// }





export const api = {
  // UI can call this to prefill the Entity ID
  
  // Called by your Create form
  async createEvent(payload) {
    // payload expected:
    // { eventType, entityId, priority, recipientEmail, chaosFailPercent }
     
  const res = await fetch("http://localhost:8080/events", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data=await res.json();
  return data
//   

   
  },

  // Called by table polling
async listNotifications() {
  const res = await fetch("http://localhost:8080/notifications");
  
  return res.json();
},

async replay(task_id) {
    // called when replay button is pressed
  if (!task_id) {
    throw new Error("task_id is required for replay");
  }
  console.log(typeof(JSON.stringify(task_id)))
  console.log(task_id)
  const res = await fetch(
    `http://localhost:8080/tasks/${encodeURIComponent(task_id.task_id)}/replay`,
    {
      method: "POST",
    }
  );

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "Replay request failed");
  }

 
  try {
    return await res.json();
  } catch {
    return {};
  }
}


};
