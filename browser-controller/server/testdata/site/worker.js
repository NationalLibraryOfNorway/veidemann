self.addEventListener("message", async () => {
  try {
    const response = await fetch("/worker-data.json", { cache: "no-store" });
    const payload = await response.json();
    self.postMessage({ outlink: payload.outlink });
    self.close();
  } catch (error) {
    self.postMessage({ error: String(error) });
    self.close();
  }
});