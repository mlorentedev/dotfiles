import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  // Listen for the final actionable boundary before the agent settles/idles
  pi.on("agent_before_settle", async (event, ctx) => {
    const usage = ctx.getContextUsage();
    
    // Ensure context usage is available and token counts are initialized
    if (!usage || usage.tokens === null) return;

    // Calculate the dynamic trigger threshold: 40% of the active model's context window
    const triggerTokens = usage.contextWindow * 0.40;

    // Trigger compaction if we exceed the calculated threshold
    if (usage.tokens >= triggerTokens) {
      if (ctx.hasUI) {
         const percent = Math.round(usage.percent! * 100);
         ctx.ui.notify(
           `Context at ${percent}% (${usage.tokens}/${usage.contextWindow}). Triggering 40% dynamic compaction threshold.`,
           "info"
         );
      }
      
      // Request asynchronous session compaction
      ctx.compact({
        customInstructions: "Context utilization exceeded the 40% dynamic threshold."
      });
    }
  });
}
