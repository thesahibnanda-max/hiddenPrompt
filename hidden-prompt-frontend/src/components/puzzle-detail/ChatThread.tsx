import { MessageCircleQuestion } from "lucide-react";
import type { MessageElement } from "@/lib/api/types";
import { ChatBubbleUser } from "./ChatBubbleUser";
import { ChatBubbleAssistant } from "./ChatBubbleAssistant";

export function ChatThread({ messages }: { messages: MessageElement[] }) {
  if (messages.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-10 text-center text-chrome-mid">
        <MessageCircleQuestion size={28} className="text-neon-purple" />
        <p className="text-sm">No guesses yet. Swing away.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {messages.map((m, i) =>
        m.role === "user" ? (
          <ChatBubbleUser key={`${m.timestamp}-${i}`} message={m} />
        ) : (
          <ChatBubbleAssistant key={`${m.timestamp}-${i}`} message={m} />
        ),
      )}
    </div>
  );
}
