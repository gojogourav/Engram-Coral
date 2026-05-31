"use client";

import { useState, useRef, useEffect } from "react";
import {
  MessageSquare,
  Send,
  Bot,
  User,
  Loader2,
  Database,
  Terminal,
  Clock,
  Sparkles,
  ChevronRight,
  Info,
} from "lucide-react";

// --- Types ---
interface Message {
  role: "user" | "assistant" | "system" | "error";
  content: string;
  ts?: string;
}

const SUGGESTED_PROMPTS = [
  "Who authored the commit that triggered the latest failed workflow on engram-test-repo?",
  "Are there any alerts currently in a 'firing' state in Grafana?",
  "List the last 5 commits on engram-test-repo.",
  "Look at the last 3 failed jobs. Are they failing on the same step?",
];

const AVAILABLE_SCHEMAS = [
  {
    source: "github",
    tables: [
      "commits",
      "jobs",
      "workflows",
      "repo_action_workflow_runs",
      "contents",
    ],
  },
  {
    source: "grafana",
    tables: ["alerts", "dashboards", "datasources"],
  },
];

export default function AgentChatPage() {
  const [chatInput, setChatInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [messages, setMessages] = useState<Message[]>([
    {
      role: "assistant",
      content:
        "⚡ Engram SRE Agent online. I am connected to the Coral Data Fabric. Ask me to investigate CI failures, query telemetry, or analyze repository states.",
      ts: new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      }),
    },
  ]);

  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isLoading]);

  const handleSend = async (e?: React.FormEvent, overrideText?: string) => {
    e?.preventDefault();
    const query = (overrideText ?? chatInput).trim();
    if (!query || isLoading) return;

    const ts = new Date().toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
    setMessages((prev) => [...prev, { role: "user", content: query, ts }]);
    setChatInput("");
    setIsLoading(true);

    try {
      const res = await fetch("http://127.0.0.1:8080/api/agent/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });

      const data = await res.json();
      const endTs = new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });

      if (data.error) {
        setMessages((prev) => [
          ...prev,
          { role: "error", content: data.error, ts: endTs },
        ]);
      } else {
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant",
            content: data.result || "No data returned.",
            ts: endTs,
          },
        ]);
      }
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          role: "error",
          content: `Network or Server Error: ${err.message}`,
          ts,
        },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <main className="min-h-screen bg-[#fbfbfa] text-[#37352f] font-sans selection:bg-[#2383e2]/20 flex flex-col">
      {/* Top Navigation / Header */}
      <header className="bg-white border-b border-[#e9e9e6] px-6 py-4 flex items-center justify-between flex-shrink-0 z-10 sticky top-0">
        <div className="flex items-center gap-3">
          <div className="bg-[#eaf2ff] p-2 rounded-md border border-[#cce0ff]">
            <Terminal className="w-5 h-5 text-[#0c3d70]" />
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight text-[#37352f] flex items-center gap-2">
              Engram ChatOps
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
              </span>
            </h1>
            <p className="text-xs text-[#787774]">
              Autonomous reasoning via Coral Data Fabric
            </p>
          </div>
        </div>
      </header>

      {/* Main Workspace */}
      <div className="flex-grow flex overflow-hidden">
        {/* Left Side: Chat Feed & Input */}
        <div className="flex-grow flex flex-col max-w-5xl border-r border-[#e9e9e6] bg-white shadow-[4px_0_24px_rgba(0,0,0,0.02)] z-0">
          {/* Scrollable Message List */}
          <div
            ref={scrollRef}
            className="flex-grow overflow-y-auto p-6 space-y-6 scrollbar-thin scrollbar-thumb-zinc-200"
          >
            {messages.map((msg, idx) => (
              <div
                key={idx}
                className={`flex gap-4 ${msg.role === "user" ? "justify-end" : "justify-start"}`}
              >
                {/* Assistant/Error Avatar */}
                {msg.role !== "user" && (
                  <div className="flex-shrink-0 mt-1">
                    <div
                      className={`p-1.5 rounded-sm border ${msg.role === "error" ? "bg-[#fdebec] border-[#fad6d8] text-[#c1292e]" : "bg-[#f1f1ef] border-[#e9e9e6] text-[#37352f]"}`}
                    >
                      <Bot className="w-4 h-4" />
                    </div>
                  </div>
                )}

                {/* Message Bubble */}
                <div
                  className={`max-w-[85%] ${msg.role === "user" ? "bg-[#f1f1ef] border border-[#e9e9e6] rounded-2xl rounded-tr-sm px-4 py-3" : "pt-1"}`}
                >
                  {/* Meta tag for Agent responses */}
                  {msg.role !== "user" && (
                    <div className="flex items-center gap-2 mb-1 text-[11px] font-medium text-[#787774] font-mono">
                      <span>
                        {msg.role === "error" ? "SYSTEM ERROR" : "ENGRAM AGENT"}
                      </span>
                      {msg.ts && <span>· {msg.ts}</span>}
                    </div>
                  )}

                  {/* Content Formatting */}
                  <div
                    className={`text-sm leading-relaxed whitespace-pre-wrap break-words ${msg.role === "error" ? "text-[#c1292e] font-mono" : "text-[#37352f]"}`}
                  >
                    {msg.role !== "user" ? (
                      // Apply slight markdown styling for agent responses (Markdown tables/code blocks)
                      <div
                        className="font-mono text-[13px] overflow-x-auto scrollbar-hide"
                        dangerouslySetInnerHTML={{
                          __html: msg.content
                            .replace(
                              /```([\s\S]*?)```/g,
                              '<pre class="bg-[#fbfbfa] p-3 rounded border border-[#e9e9e6] my-2 overflow-x-auto">$1</pre>',
                            )
                            .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>"),
                        }}
                      />
                    ) : (
                      <span className="font-medium">{msg.content}</span>
                    )}
                  </div>

                  {/* Meta tag for User responses */}
                  {msg.role === "user" && msg.ts && (
                    <div className="text-[10px] text-[#787774] mt-1 text-right">
                      {msg.ts}
                    </div>
                  )}
                </div>

                {/* User Avatar */}
                {msg.role === "user" && (
                  <div className="flex-shrink-0 mt-1">
                    <div className="p-1.5 bg-[#2383e2]/10 border border-[#2383e2]/20 rounded-sm text-[#2383e2]">
                      <User className="w-4 h-4" />
                    </div>
                  </div>
                )}
              </div>
            ))}

            {/* Loading State */}
            {isLoading && (
              <div className="flex gap-4 justify-start">
                <div className="flex-shrink-0 mt-1">
                  <div className="p-1.5 rounded-sm bg-[#f1f1ef] border border-[#e9e9e6] text-[#37352f]">
                    <Bot className="w-4 h-4" />
                  </div>
                </div>
                <div className="pt-2 flex items-center gap-2 text-xs font-mono text-[#787774]">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  <span>Agent is reasoning and executing SQL...</span>
                </div>
              </div>
            )}
          </div>

          {/* Input Area */}
          <div className="p-4 bg-white border-t border-[#e9e9e6]">
            <form
              onSubmit={handleSend}
              className="relative flex items-end gap-2 bg-[#fbfbfa] border border-[#e9e9e6] focus-within:border-[#2383e2] focus-within:ring-1 focus-within:ring-[#2383e2]/20 rounded-lg p-2 transition-all shadow-sm"
            >
              <textarea
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    handleSend();
                  }
                }}
                placeholder="Ask Engram to investigate an issue or query infrastructure..."
                className="flex-grow max-h-[200px] min-h-[44px] text-sm py-2 px-2 bg-transparent text-[#37352f] placeholder-[#c4c4c2] outline-none resize-none font-sans"
                rows={1}
                disabled={isLoading}
              />
              <button
                type="submit"
                disabled={!chatInput.trim() || isLoading}
                className="mb-1 mr-1 bg-[#2383e2] text-white disabled:bg-[#f1f1ef] disabled:text-[#c4c4c2] p-2 rounded-md transition-colors"
              >
                <Send className="w-4 h-4" />
              </button>
            </form>
            <div className="flex items-center justify-between mt-2 px-2 text-[10px] text-[#787774]">
              <span>
                Press{" "}
                <kbd className="font-mono bg-[#f1f1ef] border border-[#e9e9e6] rounded px-1 py-0.5">
                  Enter
                </kbd>{" "}
                to send,{" "}
                <kbd className="font-mono bg-[#f1f1ef] border border-[#e9e9e6] rounded px-1 py-0.5">
                  Shift + Enter
                </kbd>{" "}
                for new line.
              </span>
              <span className="flex items-center gap-1">
                <Sparkles className="w-3 h-3" /> Gemini 1.5 Flash
              </span>
            </div>
          </div>
        </div>

        {/* Right Side: Context & References Sidebar */}
        <div className="hidden lg:flex flex-col w-80 bg-[#fbfbfa] flex-shrink-0 overflow-y-auto">
          {/* Quick Actions */}
          <div className="p-6 border-b border-[#e9e9e6]">
            <h3 className="text-xs font-bold text-[#37352f] uppercase tracking-wider mb-3 flex items-center gap-1.5">
              <MessageSquare className="w-4 h-4" /> Suggested Queries
            </h3>
            <div className="space-y-2">
              {SUGGESTED_PROMPTS.map((prompt, i) => (
                <button
                  key={i}
                  onClick={() => handleSend(undefined, prompt)}
                  disabled={isLoading}
                  className="w-full text-left text-xs text-[#37352f] bg-white border border-[#e9e9e6] p-2.5 rounded hover:bg-[#f1f1ef] hover:border-[#d9d9d6] transition-colors leading-relaxed disabled:opacity-50"
                >
                  {prompt}
                </button>
              ))}
            </div>
          </div>

          {/* Schema Explorer */}
          <div className="p-6">
            <h3 className="text-xs font-bold text-[#37352f] uppercase tracking-wider mb-3 flex items-center gap-1.5">
              <Database className="w-4 h-4" /> Active Fabric Schema
            </h3>
            <div className="space-y-4">
              {AVAILABLE_SCHEMAS.map((schema) => (
                <div
                  key={schema.source}
                  className="bg-white border border-[#e9e9e6] rounded overflow-hidden"
                >
                  <div className="bg-[#f1f1ef] px-3 py-2 text-xs font-bold text-[#37352f] border-b border-[#e9e9e6] flex items-center justify-between">
                    {schema.source}
                    <span className="text-[10px] font-normal text-[#787774] bg-white px-1.5 py-0.5 rounded border border-[#e9e9e6]">
                      {schema.tables.length} tables
                    </span>
                  </div>
                  <div className="p-2 flex flex-wrap gap-1.5">
                    {schema.tables.map((table) => (
                      <span
                        key={table}
                        className="text-[10px] font-mono text-[#787774] bg-[#fbfbfa] border border-[#e9e9e6] px-1.5 py-0.5 rounded"
                      >
                        {table}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-4 p-3 bg-[#eaf2ff] border border-[#cce0ff] rounded-md flex items-start gap-2">
              <Info className="w-4 h-4 text-[#0c3d70] flex-shrink-0 mt-0.5" />
              <p className="text-[11px] text-[#0c3d70] leading-relaxed">
                The agent translates natural language into Steampipe/Coral SQL
                to query these tables in real-time. It will automatically
                correct its own syntax errors.
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
