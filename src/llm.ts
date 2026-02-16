export interface ChatMessage {
  role: string;
  content: string;
}

export interface ChatResponse {
  role: string;
  content: string;
  reasoning: string;
}

export interface LLMConfig {
  baseUrl: string;
  apiKey: string;
  modelId: string;
  reasoningEnabled?: boolean;
}

export interface StreamCallbacks {
  onContent?: (chunk: string) => void;
  onReasoning?: (chunk: string) => void;
}

export async function chatCompletion(
  history: ChatMessage[],
  config: LLMConfig,
  callbacks?: StreamCallbacks,
): Promise<ChatResponse> {
  const baseUrl = config.baseUrl || "https://api.openai.com/v1";
  const model = config.modelId || "gpt-3.5-turbo";

  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const response = await fetch(`${baseUrl}/chat/completions`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${config.apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          model,
          messages: history,
          stream: true,
          ...(config.reasoningEnabled
            ? { reasoning: { enabled: true, exclude: false } }
            : { reasoning: { enabled: false, exclude: false } }),
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`API Error: ${errorText}`);
      }

      const contentType = response.headers.get("content-type") || "";
      if (!contentType.includes("text/event-stream") && !contentType.includes("application/json")) {
        throw new Error(`Unexpected response type: ${contentType}. Check your Base URL.`);
      }

      const reader = response.body!.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let content = "";
      let reasoning = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const raw = decoder.decode(value, { stream: true });
        buffer += raw;
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed || !trimmed.startsWith("data: ")) continue;
          const data = trimmed.slice(6);
          if (data === "[DONE]") continue;

          try {
            const parsed = JSON.parse(data);
            const delta = parsed.choices?.[0]?.delta;
            if (delta?.content) {
              content += delta.content;
              callbacks?.onContent?.(delta.content);
            }
            if (delta?.reasoning_content) {
              reasoning += delta.reasoning_content;
              callbacks?.onReasoning?.(delta.reasoning_content);
            } else if (delta?.reasoning) {
              reasoning += delta.reasoning;
              callbacks?.onReasoning?.(delta.reasoning);
            }
          } catch {
            // skip malformed JSON chunks
          }
        }
      }

      return { role: "assistant", content, reasoning };
    } catch (e) {
      lastError = e;
      console.error(`chatCompletion attempt ${attempt + 1}/3 failed:`, e);
      if (attempt < 2) await new Promise(r => setTimeout(r, 1000 * (attempt + 1)));
    }
  }

  throw lastError;
}
