"use client";

import { useState, useRef, useEffect } from "react";
import {
  Activity,
  MessageSquare,
  Send,
  Bot,
  User,
  Loader2,
  ShieldAlert,
  TerminalSquare,
  CheckCircle2,
  ArrowUpCircle,
  RotateCcw,
  Server,
  AlertCircle,
  Database,
  Wifi,
  WifiOff,
  Cpu,
  HardDrive,
  Zap,
  ChevronRight,
  RefreshCw,
  Circle,
  Sparkles,
} from "lucide-react";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Message {
  role: "user" | "assistant" | "system";
  content: string;
  ts?: string;
}

interface Stats {
  webhooks_total: number;
  fixes_generated: number;
  prs_opened: number;
  success_rate: number;
  avg_ai_latency: number;
  diff_errors: number;
  fixes_failed: number;
  status: "synced" | "offline" | "connecting...";
}

interface K8sHealth {
  status: "ok" | "error" | "unknown" | "checking";
  message: string;
  deployments?: number;
  pods?: number;
}

// ─── Suggestion sets ──────────────────────────────────────────────────────────

const CORAL_SUGGESTIONS = [
  {
    label: "Recent commits",
    query:
      "SELECT sha, author__login, commit__message FROM github.commits WHERE owner = 'gojogourav' AND repo = 'engram-test-repo' LIMIT 5",
  },
  {
    label: "List workflows",
    query:
      "SELECT id, name, state FROM github.workflows WHERE owner = 'gojogourav' AND repo = 'engram-test-repo' LIMIT 5",
  },
  {
    label: "User repos",
    query: "SELECT full_name, private FROM github.user_repos LIMIT 10",
  },
  {
    label: "Search issues",
    query:
      "SELECT number, title, state FROM github.search_issues(q => 'repo:gojogourav/engram-test-repo is:issue') LIMIT 5",
  },
];
const CHAT_SUGGESTIONS = [
  { label: "List pods", cmd: "/k8s list all pods" },
  { label: "Crashing pods", cmd: "/k8s show crashing pods" },
  { label: "Deployments", cmd: "/k8s list deployments" },
  { label: "Grafana status", cmd: "/grafana summary" },
  { label: "View incident", cmd: "/incident" },
  { label: "Approve fix", cmd: "/approve" },
];

// ─── Helpers ──────────────────────────────────────────────────────────────────

const BASE = "http://127.0.0.1:8080";

async function postJSON(endpoint: string, body: object): Promise<any> {
  const res = await fetch(`${BASE}${endpoint}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) throw new Error(text || `HTTP ${res.status}`);
  if (!text.trim()) return {};
  try {
    return JSON.parse(text);
  } catch {
    return { result: text };
  }
}

function formatCoralResult(raw: string): {
  type: "json" | "error" | "text";
  content: string;
} {
  const trimmed = raw.trim();
  if (
    trimmed.includes("panicked at") ||
    trimmed.startsWith("Error:") ||
    trimmed.startsWith("thread '")
  ) {
    const errorLine = trimmed
      .split("\n")
      .find(
        (l) =>
          l.startsWith("Error:") ||
          l.startsWith("Detail:") ||
          l.startsWith("Hint:"),
      );
    return { type: "error", content: errorLine || trimmed };
  }
  try {
    const parsed = JSON.parse(trimmed);
    return { type: "json", content: JSON.stringify(parsed, null, 2) };
  } catch {
    return { type: "text", content: trimmed };
  }
}

function timestamp() {
  return new Date().toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function StatusDot({ ok, pulse }: { ok: boolean; pulse?: boolean }) {
  return (
    <span className="relative flex h-2.5 w-2.5">
      {pulse && ok && (
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
      )}
      <span
        className={`relative inline-flex rounded-full h-2.5 w-2.5 ${ok ? "bg-emerald-500" : "bg-rose-500"}`}
      ></span>
    </span>
  );
}

function K8sHealthBadge({
  health,
  onCheck,
}: {
  health: K8sHealth;
  onCheck: () => void;
}) {
  const color =
    health.status === "ok"
      ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400"
      : health.status === "error"
        ? "bg-rose-500/10 border-rose-500/20 text-rose-400"
        : health.status === "checking"
          ? "bg-amber-500/10 border-amber-500/20 text-amber-400"
          : "bg-zinc-800/50 border-zinc-700/50 text-zinc-400";

  return (
    <div
      className={`flex items-center gap-2 px-3 py-1.5 rounded-full border text-xs font-medium backdrop-blur-sm transition-colors ${color}`}
    >
      {health.status === "checking" ? (
        <Loader2 className="w-3.5 h-3.5 animate-spin" />
      ) : health.status === "ok" ? (
        <CheckCircle2 className="w-3.5 h-3.5" />
      ) : health.status === "error" ? (
        <WifiOff className="w-3.5 h-3.5" />
      ) : (
        <Circle className="w-3.5 h-3.5" />
      )}
      <span>{health.message}</span>
      {health.deployments !== undefined && (
        <span className="opacity-60 hidden sm:inline">
          · {health.deployments} deploys
        </span>
      )}
      <button
        onClick={onCheck}
        className="ml-1 opacity-50 hover:opacity-100 transition-opacity focus:outline-none"
        title="Re-check K8s"
      >
        <RefreshCw className="w-3 h-3" />
      </button>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function ChatOpsDashboard() {
  const [stats, setStats] = useState<Stats>({
    webhooks_total: 0,
    fixes_generated: 0,
    prs_opened: 0,
    success_rate: 0,
    avg_ai_latency: 0,
    diff_errors: 0,
    fixes_failed: 0,
    status: "connecting...",
  });

  const [incidents, setIncidents] = useState<any[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [isChatLoading, setIsChatLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"chat" | "coral">("chat");
  const [chatMessages, setChatMessages] = useState<Message[]>([
    {
      role: "assistant",
      content:
        "⚡ Engram SRE online. Use /k8s, /docker, /grafana, /approve, /incident commands — or pick a suggestion above.",
      ts: timestamp(),
    },
  ]);

  const [k8sHealth, setK8sHealth] = useState<K8sHealth>({
    status: "unknown",
    message: "Not checked",
  });

  const chatScrollRef = useRef<HTMLDivElement>(null);

  // ── Polling ────────────────────────────────────────────────────────────────

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await fetch(`${BASE}/api/stats`);
        if (res.ok) setStats({ ...(await res.json()), status: "synced" });
      } catch {
        setStats((p) => ({ ...p, status: "offline" }));
      }
    };
    fetchStats();
    const iv = setInterval(fetchStats, 2500);
    return () => clearInterval(iv);
  }, []);

  useEffect(() => {
    const fetchIncidents = async () => {
      try {
        const res = await fetch(`${BASE}/api/incidents`);
        if (res.ok) {
          const data = await res.json();
          setIncidents(
            (data || []).sort(
              (a: any, b: any) =>
                new Date(b.created_at).getTime() -
                new Date(a.created_at).getTime(),
            ),
          );
        }
      } catch {}
    };
    fetchIncidents();
    const iv = setInterval(fetchIncidents, 2500);
    return () => clearInterval(iv);
  }, []);

  useEffect(() => {
    if (chatScrollRef.current)
      chatScrollRef.current.scrollTop = chatScrollRef.current.scrollHeight;
  }, [chatMessages]);

  // ── K8s health check ───────────────────────────────────────────────────────

  const checkK8sHealth = async () => {
    setK8sHealth({ status: "checking", message: "Connecting to cluster…" });
    try {
      const res = await fetch(`${BASE}/k8s/command`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text: "list deployments" }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const text = await res.text();
      let result = text;
      try {
        result = JSON.parse(text)?.result ?? text;
      } catch {}
      if (!result && text) result = text;
      const deploys = (result.match(/Deployment:/g) || []).length;
      setK8sHealth({
        status: "ok",
        message: deploys > 0 ? "Cluster Online" : "Cluster Reachable",
        deployments: deploys,
        pods: undefined,
      });
    } catch (e: any) {
      setK8sHealth({
        status: "error",
        message: e.message?.slice(0, 40) || "Cluster unreachable",
      });
    }
  };

  useEffect(() => {
    checkK8sHealth();
  }, []);

  // ── Message helpers ────────────────────────────────────────────────────────

  const addMsg = (role: Message["role"], content: string) =>
    setChatMessages((p) => [...p, { role, content, ts: timestamp() }]);

  // ── Send message ───────────────────────────────────────────────────────────

  const handleSend = async (e?: React.FormEvent, overrideInput?: string) => {
    e?.preventDefault();
    const command = (overrideInput ?? chatInput).trim();
    if (!command) return;
    addMsg("user", command);
    setChatInput("");
    setIsChatLoading(true);

    try {
      if (command.startsWith("/k8s")) {
        const text = command.replace("/k8s", "").trim();
        const data = await postJSON("/k8s/command", { text });
        const result = (data.result ?? data.error ?? data.message ?? "").trim();
        if (!result) {
          addMsg(
            "assistant",
            `⚠️ K8s returned empty for: "${text}"\n\nThe Go K8sCommandHandler is currently a stub.`,
          );
        } else {
          addMsg("assistant", result);
        }
      } else if (command.startsWith("/docker")) {
        const text = command.replace("/docker", "").trim();
        const data = await postJSON("/docker/command", { text });
        const result = (data.result ?? data.error ?? data.message ?? "").trim();
        addMsg("assistant", result || `⚠️ Docker returned empty.`);
      } else if (command.startsWith("/grafana")) {
        const text = command.replace("/grafana", "").trim();
        const data = await postJSON("/grafana/command", { text });
        const result = (data.result ?? data.error ?? data.message ?? "").trim();
        addMsg("assistant", result || `⚠️ Grafana returned empty.`);
      } else if (command.startsWith("/approve")) {
        const id = command.replace("/approve", "").trim();
        const target =
          id || incidents.find((i) => i.stage === "pending_approval")?.id;
        if (!target) {
          addMsg("assistant", "ℹ️ No incident is currently pending approval.");
        } else {
          await postJSON("/api/approve", { id: target });
          addMsg("assistant", `✅ Approved incident **${target}**.`);
        }
      } else if (command.startsWith("/incident")) {
        const keyword = command.replace("/incident", "").trim();
        const match =
          incidents.find((i) => i.id.includes(keyword)) || incidents[0];
        if (!match) {
          addMsg("assistant", "ℹ️ No incidents found.");
        } else {
          addMsg(
            "assistant",
            `📋 **${match.id}**\nRepo: ${match.repo}\nStage: ${match.stage}\nConfidence: ${match.confidence_score}%\nFailed Jobs:\n${match.failed_jobs || "N/A"}${match.pr_url ? `\nPR: ${match.pr_url}` : ""}`,
          );
        }
      } else {
        addMsg(
          "assistant",
          `❓ Unknown command. Try:\n/k8s <action>\n/docker <action>\n/grafana <query>\n/approve [id]\n/incident [id]\n\nOr switch to the Coral tab for live SQL.`,
        );
      }
    } catch (err: any) {
      addMsg("assistant", `❌ ${err.message}`);
    }
    setIsChatLoading(false);
  };

  // ── Coral query ────────────────────────────────────────────────────────────

  const runCoralQuery = async (query: string) => {
    addMsg("user", query);
    setIsChatLoading(true);
    try {
      const res = await fetch(`${BASE}/coral/query`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });
      const raw = await res.text();
      const fmt = formatCoralResult(raw);
      if (!res.ok || fmt.type === "error") {
        addMsg("assistant", `⚠️ Coral Error:\n${fmt.content}`);
        setIsChatLoading(false);
        return;
      }
      if (fmt.type === "json") {
        try {
          const arr = JSON.parse(fmt.content);
          if (Array.isArray(arr) && arr.length === 0) {
            addMsg("assistant", "✅ Query succeeded — no rows returned.");
          } else if (Array.isArray(arr)) {
            const cols = Object.keys(arr[0]);
            const colW = cols.map((c) =>
              Math.max(
                c.length,
                ...arr.slice(0, 8).map((r: any) => String(r[c] ?? "").length),
                8,
              ),
            );
            const pad = (s: string, w: number) => s.slice(0, w).padEnd(w);
            const header = cols.map((c, i) => pad(c, colW[i])).join(" │ ");
            const divider = colW.map((w) => "─".repeat(w)).join("─┼─");
            const rows = arr
              .slice(0, 8)
              .map((row: any) =>
                cols
                  .map((c, i) => pad(String(row[c] ?? ""), colW[i]))
                  .join(" │ "),
              )
              .join("\n");
            const suffix =
              arr.length > 8 ? `\n… +${arr.length - 8} more rows` : "";
            addMsg(
              "assistant",
              `📊 ${arr.length} row(s):\n\`\`\`\n${header}\n${divider}\n${rows}${suffix}\n\`\`\``,
            );
          } else {
            addMsg(
              "assistant",
              `📊 Coral Output:\n\`\`\`json\n${fmt.content}\n\`\`\``,
            );
          }
        } catch {
          addMsg("assistant", `📊 Coral:\n\`\`\`json\n${fmt.content}\n\`\`\``);
        }
      } else {
        addMsg("assistant", `📄 ${fmt.content}`);
      }
    } catch (err: any) {
      addMsg("assistant", `❌ Coral error: ${err.message}`);
    }
    setIsChatLoading(false);
  };

  const firstDeployment = (): string => {
    const msgs = chatMessages.filter((m) => m.role === "assistant");
    for (const m of [...msgs].reverse()) {
      const match = m.content.match(/Deployment:\s*([\w-]+)/);
      if (match) return match[1];
    }
    return "broken-app";
  };

  const executeK8sAction = async (label: string, cmd: string) => {
    await handleSend(undefined, `/k8s ${cmd}`);
  };

  const pendingIncident = incidents.find((i) => i.stage === "pending_approval");

  return (
    <main className="min-h-screen bg-white text-[#37352f] p-4 sm:p-6 lg:p-8 selection:bg-[#2383e2]/20 font-sans">
      <div className="max-w-7xl mx-auto space-y-6 pb-20">
        {/* Pending Approval Callout Block */}
        {pendingIncident && (
          <div className="bg-[#faebdd] border border-[#e9e9e6] rounded-md p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 shadow-sm animate-in fade-in duration-200">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-[#69311c] mt-0.5 flex-shrink-0" />
              <div>
                <h3 className="text-sm font-semibold text-[#69311c]">
                  Action Required: {pendingIncident.id}
                </h3>
                <p className="text-xs text-[#69311c]/80 mt-0.5">
                  AI Fix is awaiting your approval ·{" "}
                  {pendingIncident.confidence_score}% Confidence
                </p>
              </div>
            </div>
            <button
              onClick={() =>
                postJSON("/api/approve", { id: pendingIncident.id })
              }
              className="bg-white hover:bg-[#f1f1ef] text-[#69311c] border border-[#d9cbbf] px-3 py-1.5 rounded text-xs font-medium transition-colors shadow-sm active:bg-[#e9e9e6] flex items-center justify-center gap-1 whitespace-nowrap"
            >
              Approve Deployment <ChevronRight className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Header Section */}
        <header className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-[#e9e9e6]">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <ShieldAlert className="w-6 h-6 text-[#37352f]" />
              <h1 className="text-2xl md:text-3xl font-bold tracking-tight text-[#37352f]">
                Command Center
              </h1>
            </div>
            <p className="text-[#787774] text-xs md:text-sm pl-8">
              Live SRE telemetry · Kubernetes orchestration · Data Fabric
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2 pl-8 md:pl-0">
            <K8sHealthBadge health={k8sHealth} onCheck={checkK8sHealth} />
            <div
              className={`flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded border transition-colors ${
                stats.status === "synced"
                  ? "bg-[#e2f2e4] border-[#d2e2d4] text-[#1c4d2d]"
                  : "bg-[#f1f1ef] border-[#e9e9e6] text-[#787774]"
              }`}
            >
              <StatusDot ok={stats.status === "synced"} pulse={false} />
              {stats.status === "synced"
                ? "Backend Synced"
                : stats.status === "connecting..."
                  ? "Connecting…"
                  : "Backend Offline"}
            </div>
          </div>
        </header>

        {/* Gallery-Style Metrics Grid */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            {
              label: "Failures Caught",
              value: stats.webhooks_total,
              icon: <Zap className="w-4 h-4 text-[#6e2c2c]" />,
              bg: "bg-[#fdebec]",
              color: "text-[#6e2c2c]",
            },
            {
              label: "Fixes Generated",
              value: stats.fixes_generated,
              icon: <Bot className="w-4 h-4 text-[#0c3d70]" />,
              bg: "bg-[#eaf2ff]",
              color: "text-[#0c3d70]",
            },
            {
              label: "Success Rate",
              value: `${Number(stats.success_rate || 0).toFixed(1)}%`,
              icon: <Sparkles className="w-4 h-4 text-[#1c4d2d]" />,
              bg: "bg-[#e2f2e4]",
              color: "text-[#1c4d2d]",
            },
            {
              label: "Avg AI Latency",
              value: `${Number(stats.avg_ai_latency || 0).toFixed(1)}s`,
              icon: <Activity className="w-4 h-4 text-[#69311c]" />,
              bg: "bg-[#faebdd]",
              color: "text-[#69311c]",
            },
          ].map((m) => (
            <div
              key={m.label}
              className="bg-white border border-[#e9e9e6] rounded-md p-4 transition-colors hover:bg-[#fbfbfa]"
            >
              <div className="flex items-center gap-1.5 text-[#787774] text-xs font-normal mb-1.5">
                <span className={`p-1 rounded ${m.bg}`}>{m.icon}</span>
                {m.label}
              </div>
              <div
                className={`text-xl font-mono font-semibold tracking-tight ${m.color}`}
              >
                {m.value}
              </div>
            </div>
          ))}
        </div>

        {/* Main Interface Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* ── View Workspace Container (Left) ── */}
          <div className="lg:col-span-8 bg-white border border-[#e9e9e6] rounded-md flex flex-col h-[650px] overflow-hidden shadow-sm">
            {/* Notion Database-Style Views Tab Bar */}
            <div className="border-b border-[#e9e9e6] px-3 bg-[#fbfbfa] flex items-center justify-between gap-3 min-h-[40px]">
              <div className="flex gap-1">
                <button
                  onClick={() => setActiveTab("chat")}
                  className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
                    activeTab === "chat"
                      ? "border-[#37352f] text-[#37352f]"
                      : "border-transparent text-[#787774] hover:bg-[#f1f1ef] hover:text-[#37352f]"
                  }`}
                >
                  <MessageSquare className="w-3.5 h-3.5" /> Terminal view
                </button>
                <button
                  onClick={() => setActiveTab("coral")}
                  className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
                    activeTab === "coral"
                      ? "border-[#37352f] text-[#37352f]"
                      : "border-transparent text-[#787774] hover:bg-[#f1f1ef] hover:text-[#37352f]"
                  }`}
                >
                  <Database className="w-3.5 h-3.5" /> Data Fabric view
                </button>
              </div>
            </div>

            {/* Action Tags Bar */}
            <div className="border-b border-[#e9e9e6] bg-[#fbfbfa]/50 px-4 py-2 flex gap-1.5 overflow-x-auto scrollbar-hide">
              {activeTab === "chat"
                ? CHAT_SUGGESTIONS.map((s) => (
                    <button
                      key={s.cmd}
                      onClick={() => setChatInput(s.cmd)}
                      className="text-[11px] font-normal px-2.5 py-1 bg-white border border-[#e9e9e6] rounded text-[#37352f] hover:bg-[#f1f1ef] transition-colors flex-shrink-0"
                    >
                      {s.label}
                    </button>
                  ))
                : CORAL_SUGGESTIONS.map((s) => (
                    <button
                      key={s.label}
                      onClick={() => runCoralQuery(s.query)}
                      className="text-[11px] font-normal px-2.5 py-1 bg-white border border-[#e9e9e6] rounded text-[#37352f] hover:bg-[#f1f1ef] transition-colors flex-shrink-0"
                    >
                      {s.label}
                    </button>
                  ))}
            </div>

            {/* K8s Inline Context Actions */}
            {activeTab === "chat" && (
              <div className="border-b border-[#e9e9e6] bg-[#e2f2e4]/30 px-4 py-2 flex items-center gap-1.5 overflow-x-auto">
                <span className="text-[10px] text-[#1c4d2d] font-mono uppercase tracking-wider mr-1 flex-shrink-0">
                  K8s Context:
                </span>
                {[
                  { label: "List Pods", cmd: "list all pods" },
                  { label: "Crashing", cmd: "show crashing pods" },
                  { label: "Deploys", cmd: "list deployments" },
                  { label: `Restart`, cmd: `restart ${firstDeployment()}` },
                  {
                    label: `Scale ×3`,
                    cmd: `scale ${firstDeployment()} to 3 replicas`,
                  },
                ].map((s) => (
                  <button
                    key={s.cmd}
                    onClick={() => handleSend(undefined, `/k8s ${s.cmd}`)}
                    className="text-[10px] px-2 py-0.5 bg-white border border-[#d2e2d4] rounded text-[#1c4d2d] hover:bg-[#e2f2e4] transition-all font-mono flex-shrink-0"
                  >
                    {s.label}
                  </button>
                ))}
              </div>
            )}

            {/* Clean Line-Based Discussion Feed */}
            <div
              ref={chatScrollRef}
              className="flex-grow p-4 overflow-y-auto space-y-4 bg-white scrollbar-thin scrollbar-thumb-zinc-200"
            >
              {chatMessages.map((msg, i) => (
                <div
                  key={i}
                  className={`flex flex-col gap-1 w-full max-w-full ${
                    msg.role === "system"
                      ? "items-center py-2"
                      : "items-start border-l-2 pl-3 ml-1"
                  } ${
                    msg.role === "user"
                      ? "border-l-[#2383e2]"
                      : "border-l-[#e9e9e6]"
                  }`}
                >
                  {msg.role === "system" ? (
                    <div className="text-[11px] text-[#787774] font-mono bg-[#f1f1ef] px-3 py-1 rounded">
                      {msg.content}
                    </div>
                  ) : (
                    <>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-[#787774]">
                        {msg.role === "user" ? (
                          <>
                            <User className="w-3.5 h-3.5 text-[#2383e2]" />
                            <span className="text-[#37352f]">User Input</span>
                          </>
                        ) : (
                          <>
                            <Bot className="w-3.5 h-3.5 text-[#787774]" />
                            <span className="text-[#37352f]">System Agent</span>
                          </>
                        )}
                        {msg.ts && (
                          <span className="text-[10px] text-[#787774]/60 font-mono font-normal">
                            · {msg.ts}
                          </span>
                        )}
                      </div>
                      <div className="text-sm font-mono text-[#37352f] whitespace-pre-wrap break-words leading-relaxed mt-0.5">
                        {msg.content}
                      </div>
                    </>
                  )}
                </div>
              ))}
              {isChatLoading && (
                <div className="flex items-center gap-2 pl-3 ml-1 border-l-2 border-l-transparent text-xs text-[#787774] font-mono">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  {activeTab === "coral"
                    ? "Executing query..."
                    : "Analyzing..."}
                </div>
              )}
            </div>

            {/* Input Text Block */}
            <div className="p-3 bg-[#fbfbfa] border-t border-[#e9e9e6]">
              <form
                onSubmit={
                  activeTab === "coral"
                    ? (e) => {
                        e.preventDefault();
                        if (chatInput.trim()) {
                          runCoralQuery(chatInput.trim());
                          setChatInput("");
                        }
                      }
                    : handleSend
                }
                className="flex items-center gap-2 bg-white border border-[#e9e9e6] focus-within:border-[#2383e2] rounded px-2.5 py-1 transition-all"
              >
                <input
                  type="text"
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  placeholder={
                    activeTab === "coral"
                      ? "SELECT * FROM github.commits..."
                      : "Type a command e.g. /k8s list pods..."
                  }
                  className="flex-grow text-sm py-1 bg-transparent text-[#37352f] placeholder-[#c4c4c2] outline-none font-mono"
                  disabled={isChatLoading}
                />
                <button
                  type="submit"
                  disabled={!chatInput.trim() || isChatLoading}
                  className="text-[#37352f] hover:bg-[#f1f1ef] disabled:opacity-30 disabled:hover:bg-transparent p-1.5 rounded transition-colors"
                >
                  <Send className="w-4 h-4" />
                </button>
              </form>
            </div>
          </div>

          {/* ── Sidebar Panels (Right) ── */}
          <div className="lg:col-span-4 flex flex-col gap-6">
            {/* Active Incidents Database View */}
            <div className="bg-white border border-[#e9e9e6] rounded-md flex flex-col shadow-sm">
              <div className="border-b border-[#e9e9e6] px-4 py-2.5 bg-[#fbfbfa] flex items-center justify-between">
                <h3 className="text-xs font-bold text-[#37352f] uppercase tracking-wider flex items-center gap-1.5">
                  <AlertCircle className="w-4 h-4 text-[#6e2c2c]" /> Active
                  Incidents
                </h3>
                <span className="bg-[#fdebec] text-[#6e2c2c] text-[11px] font-medium px-2 py-0.5 rounded border border-[#fad6d8]">
                  {incidents.length}
                </span>
              </div>

              <div className="divide-y divide-[#e9e9e6] overflow-y-auto max-h-[260px] scrollbar-thin scrollbar-thumb-zinc-200">
                {incidents.length === 0 ? (
                  <div className="p-6 text-xs text-[#787774] text-center font-mono flex flex-col items-center gap-1.5">
                    <CheckCircle2 className="w-5 h-5 text-emerald-600" />
                    All clear. No active incidents.
                  </div>
                ) : (
                  incidents.slice(0, 5).map((inc) => (
                    <div
                      key={inc.id}
                      className="px-4 py-2.5 flex items-center justify-between hover:bg-[#fbfbfa] transition-colors"
                    >
                      <div className="space-y-0.5">
                        <p className="text-xs font-mono font-semibold text-[#37352f]">
                          {inc.id}
                        </p>
                        <div className="flex items-center gap-1.5 text-[11px]">
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              inc.stage === "healed"
                                ? "bg-emerald-500"
                                : inc.stage === "failed"
                                  ? "bg-rose-500"
                                  : "bg-amber-500 animate-pulse"
                            }`}
                          />
                          <span className="text-[#787774] capitalize font-medium">
                            {inc.stage.replace("_", " ")}
                          </span>
                        </div>
                      </div>

                      {inc.stage === "pending_approval" ? (
                        <button
                          onClick={() =>
                            postJSON("/api/approve", { id: inc.id })
                          }
                          className="text-[11px] font-medium bg-[#faebdd] text-[#69311c] border border-[#d9cbbf] px-2 py-1 rounded hover:bg-[#f4dcd0] transition-colors"
                        >
                          Approve
                        </button>
                      ) : inc.stage === "healed" ? (
                        <div className="bg-[#e2f2e4] p-1 rounded border border-[#d2e2d4]">
                          <CheckCircle2 className="w-3.5 h-3.5 text-[#1c4d2d]" />
                        </div>
                      ) : (
                        <Loader2 className="w-3.5 h-3.5 text-[#787774] animate-spin" />
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Cluster Operations Navigation List */}
            <div className="bg-white border border-[#e9e9e6] rounded-md flex flex-col shadow-sm flex-grow">
              <div className="border-b border-[#e9e9e6] px-4 py-2.5 bg-[#fbfbfa]">
                <h3 className="text-xs font-bold text-[#37352f] uppercase tracking-wider flex items-center gap-1.5">
                  <TerminalSquare className="w-4 h-4 text-[#0c3d70]" /> Cluster
                  Controls
                </h3>
              </div>
              <div className="p-1.5 flex flex-col gap-0.5 overflow-y-auto">
                {[
                  {
                    label: "List Deployments",
                    cmd: "list deployments",
                    icon: <Server className="w-3.5 h-3.5" />,
                  },
                  {
                    label: "List All Pods",
                    cmd: "list all pods",
                    icon: <Cpu className="w-3.5 h-3.5" />,
                  },
                  {
                    label: `Restart App`,
                    cmd: `restart ${firstDeployment()}`,
                    icon: <RotateCcw className="w-3.5 h-3.5" />,
                  },
                  {
                    label: `Scale ×2`,
                    cmd: `scale ${firstDeployment()} to 2 replicas`,
                    icon: <ArrowUpCircle className="w-3.5 h-3.5" />,
                  },
                ].map((op) => (
                  <button
                    key={op.label}
                    onClick={() => executeK8sAction(op.label, op.cmd)}
                    className="flex items-center justify-between p-2 rounded text-[#37352f] text-xs transition-colors hover:bg-[#f1f1ef] text-left w-full"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-[#787774]">{op.icon}</span>
                      <span className="font-medium">{op.label}</span>
                    </div>
                    <ChevronRight className="w-3.5 h-3.5 text-[#c4c4c2]" />
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
