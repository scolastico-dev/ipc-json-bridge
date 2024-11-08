import { IpcBridge } from "./dist/index.mjs";

(async () => {
  const bridge = new IpcBridge();
  await bridge.start();
  console.log("Bridge started on path", bridge.getSocketPath());
  bridge.on("connect", (e) => {
    console.log("Client connected", e);
    bridge.send({ id: e.id, msg: btoa("Hello from main process") });
  });
  bridge.on("disconnect", (e) => {
    console.log("Client disconnected", e);
  });
  bridge.on("message", (m) => {
    try {
      console.log("Received message", atob(m.msg));
    } catch {}
  });
})();
