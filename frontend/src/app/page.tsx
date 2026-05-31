"use client";

import { useEffect, useState } from "react";
import {
  GitPullRequest,
  Zap,
  ShieldCheck,
  AlertTriangle,
  Database,
  ExternalLink,
  TrendingUp,
  CheckCircle2,
  Copy,
} from "lucide-react";
import Link from "next/link";

export default function MissionControl() {
  const [incidents, setIncidents] = useState<any[]>([]);

  useEffect(() => {
    const fetchIncidents = async () => {
      try {
        const res = await fetch("http://127.0.0.1:8080/api/incidents");
        if (res.ok) {
          const data = await res.json();
          if (data) {
            const sorted = data.sort(
              (a: any, b: any) =>
                new Date(b.created_at).getTime() -
                new Date(a.created_at).getTime(),
            );
            setIncidents(sorted);
          }
        }
      } catch (err) {
        // Handled silently
      }
    };
    fetchIncidents();
    const interval = setInterval(fetchIncidents, 1500);
    return () => clearInterval(interval);
  }, []);

  const healed = incidents.filter((i) => i.stage === "healed");
  const failed = incidents.filter((i) => i.stage === "failed");
  const active = incidents.filter(
    (i) => i.stage !== "healed" && i.stage !== "failed",
  );

  const healRate =
    incidents.length > 0
      ? Math.round((healed.length / incidents.length) * 100)
      : 0;
  const lastCoralQuery = incidents.find(
    (i) => i.coral_sql_query,
  )?.coral_sql_query;
  const recentPRs = incidents.filter((i) => i.pr_url).slice(0, 5);

  const activityFeed = incidents.flatMap((inc) => {
    const events = [];
    if (inc.stage === "healed") {
      const duration = Math.round(
        (new Date(inc.updated_at).getTime() -
          new Date(inc.created_at).getTime()) /
          1000,
      );
      events.push({
        id: inc.id + "-healed",
        time: inc.updated_at,
        text: `Resolved in ${duration}s`,
        entity: inc.id,
        type: "healed",
      });
    }
    if (inc.confidence_score > 0) {
      events.push({
        id: inc.id + "-gemini",
        time: inc.updated_at,
        text: `Generated fix (${inc.confidence_score}% confidence)`,
        entity: inc.id,
        type: "gemini",
      });
    }
    if (inc.failed_jobs) {
      events.push({
        id: inc.id + "-coral",
        time: inc.created_at,
        text: `Aggregated job failures`,
        entity: inc.id,
        type: "coral",
      });
    }
    if (inc.stage === "failed") {
      events.push({
        id: inc.id + "-failed",
        time: inc.updated_at,
        text: `Could not auto-fix`,
        entity: inc.id,
        type: "failed",
      });
    }
    return events;
  });

  activityFeed.sort(
    (a, b) => new Date(b.time).getTime() - new Date(a.time).getTime(),
  );

  return (
    <div className="max-w-[1080px] mx-auto px-6 sm:px-12 py-8 sm:py-12 pb-24">
      <h1 className="text-[40px] leading-tight font-bold tracking-tight text-[#37352f] mb-8">
        Fleet Overview
      </h1>

      {/* NOTION CALLOUT (Active Incident) */}
      {active.length > 0 && (
        <div className="mb-8 bg-[#fce8e6] border border-[#f5c6cb] rounded-md p-4 flex items-start sm:items-center justify-between gap-4">
          <div className="flex items-start sm:items-center gap-3">
            <span className="text-xl leading-none">🚨</span>
            <div>
              <h3 className="font-semibold text-[#c1292e] text-sm">
                Active Incident Detected: {active[0].id}
              </h3>
              <p className="text-[#c1292e]/80 text-[13px] mt-0.5 capitalize">
                Current Stage: {active[0].stage.replace("_", " ")}
              </p>
            </div>
          </div>
          <Link
            href={`/war-room?id=${active[0].id}`}
            className="bg-[#c1292e] hover:bg-[#a62227] text-white px-3 py-1.5 rounded text-[13px] font-medium transition-colors flex-shrink-0"
          >
            {active[0].stage === "pending_approval"
              ? "Review Fix →"
              : "Open War Room →"}
          </Link>
        </div>
      )}

      {/* METRICS GRID */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-10">
        <MetricCard
          title="Failures Caught"
          value={incidents.length}
          icon={<Zap className="w-4 h-4 text-[#787774]" strokeWidth={1.5} />}
        />
        <MetricCard
          title="Auto-Healed"
          value={healed.length}
          icon={
            <ShieldCheck className="w-4 h-4 text-[#787774]" strokeWidth={1.5} />
          }
        />
        <MetricCard
          title="PRs Opened"
          value={recentPRs.length}
          icon={
            <GitPullRequest
              className="w-4 h-4 text-[#787774]"
              strokeWidth={1.5}
            />
          }
        />
        <MetricCard
          title="Failed Fixes"
          value={failed.length}
          icon={
            <AlertTriangle
              className="w-4 h-4 text-[#787774]"
              strokeWidth={1.5}
            />
          }
        />

        <div className="border border-[#e9e9e7] rounded-lg p-4 flex flex-col justify-between col-span-2 lg:col-span-1 group hover:bg-[#fbfbfa] transition-colors">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-[13px] font-medium text-[#787774]">
              Auto-Heal Rate
            </h3>
            <TrendingUp className="w-4 h-4 text-[#787774]" strokeWidth={1.5} />
          </div>
          <span className="text-[28px] font-semibold tracking-tight text-[#37352f]">
            {healRate}%
          </span>
          <div className="mt-3 h-1.5 bg-[#f1f1ef] rounded-full overflow-hidden">
            <div
              className="h-full bg-[#37352f] rounded-full transition-all duration-500"
              style={{ width: `${healRate}%` }}
            />
          </div>
        </div>
      </div>

      {/* MAIN CONTENT SPLIT */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
        {/* LEFT: INCIDENTS TABLE */}
        <div className="lg:col-span-2">
          <div className="flex items-center justify-between mb-3 border-b border-[#e9e9e7] pb-2">
            <h2 className="text-base font-semibold text-[#37352f]">
              Recent Activity
            </h2>
            <Link
              href="/war-room"
              className="text-[13px] text-[#787774] hover:text-[#37352f] transition-colors flex items-center gap-1"
            >
              View War Room{" "}
              <ExternalLink className="w-3.5 h-3.5" strokeWidth={1.5} />
            </Link>
          </div>
          <div className="overflow-hidden min-h-[200px]">
            {incidents.length === 0 ? (
              <div className="py-8 text-left text-[#787774] text-sm">
                No incidents recorded yet. All systems clear.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-[13px] whitespace-nowrap">
                  <thead className="text-[#787774] border-b border-[#e9e9e7]">
                    <tr>
                      <th className="py-2.5 font-normal w-[120px]">
                        Incident ID
                      </th>
                      <th className="py-2.5 font-normal">Repository</th>
                      <th className="py-2.5 font-normal">Commit</th>
                      <th className="py-2.5 font-normal w-[100px]">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#e9e9e7]">
                    {incidents.map((inc) => (
                      <tr
                        key={inc.id || Math.random()}
                        className="hover:bg-[#fbfbfa] transition-colors"
                      >
                        <td className="py-2.5 font-mono text-[#37352f]">
                          <Link
                            href={`/war-room?id=${inc.id}`}
                            className="hover:underline hover:text-[#2383e2]"
                          >
                            {inc.id || "N/A"}
                          </Link>
                        </td>
                        <td className="py-2.5 text-[#37352f]">
                          {inc.repo || "N/A"}
                        </td>
                        <td className="py-2.5 font-mono text-[#787774]">
                          <span className="bg-[#f1f1ef] px-1.5 py-0.5 rounded text-[12px]">
                            {inc.head_sha
                              ? inc.head_sha.substring(0, 7)
                              : "N/A"}
                          </span>
                        </td>
                        <td className="py-2.5">
                          <StatusBadge stage={inc.stage || "unknown"} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        {/* RIGHT: TIMELINE */}
        <div>
          <h2 className="text-base font-semibold text-[#37352f] mb-3 border-b border-[#e9e9e7] pb-2">
            Timeline
          </h2>
          <div className="h-[250px] overflow-y-auto pr-2">
            {activityFeed.length === 0 ? (
              <div className="text-[#787774] text-[13px] py-2">
                No recent events.
              </div>
            ) : (
              <div className="space-y-4 pt-2">
                {activityFeed.map((event, index) => (
                  <div
                    key={`${event.id}-${index}`}
                    className="flex items-start gap-3 relative"
                  >
                    {index !== activityFeed.length - 1 && (
                      <div className="absolute top-4 left-[9px] w-[1.5px] h-[calc(100%+8px)] bg-[#e9e9e7]" />
                    )}
                    <div className="w-5 h-5 rounded flex items-center justify-center bg-[#f1f1ef] border border-[#e9e9e7] z-10 flex-shrink-0 mt-0.5">
                      {event.type === "healed" ? (
                        <CheckCircle2 className="w-3 h-3 text-[#1e4620]" />
                      ) : event.type === "gemini" ? (
                        <Zap className="w-3 h-3 text-[#9b51e0]" />
                      ) : event.type === "coral" ? (
                        <Database className="w-3 h-3 text-[#2383e2]" />
                      ) : (
                        <AlertTriangle className="w-3 h-3 text-[#c1292e]" />
                      )}
                    </div>
                    <div className="pb-1">
                      <p className="text-[13px] text-[#37352f] leading-snug font-medium">
                        {event.text}
                      </p>
                      <p className="text-[12px] text-[#787774] mt-0.5 flex items-center gap-1.5">
                        <span className="font-mono">{event.entity}</span>
                        <span>·</span>
                        <span>
                          {new Date(event.time).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </span>
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* BOTTOM GRID */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* CORAL QUERY INSPECTOR */}
        <div>
          <div className="flex items-center justify-between mb-3 border-b border-[#e9e9e7] pb-2">
            <h2 className="text-base font-semibold text-[#37352f] flex items-center gap-2">
              <Database className="w-4 h-4 text-[#37352f]" strokeWidth={1.5} />{" "}
              Coral Inspector
            </h2>
            {lastCoralQuery && (
              <button
                onClick={() => navigator.clipboard.writeText(lastCoralQuery)}
                className="text-[12px] flex items-center gap-1.5 text-[#787774] hover:text-[#37352f] transition-colors hover:bg-[#efefed] px-1.5 py-0.5 rounded"
              >
                <Copy className="w-3.5 h-3.5" strokeWidth={1.5} /> Copy
              </button>
            )}
          </div>
          <div className="bg-[#f7f7f5] rounded-md p-4 min-h-[120px] flex flex-col font-mono text-[13px]">
            {lastCoralQuery ? (
              <div className="text-[#37352f] overflow-x-auto flex-grow">
                <pre className="whitespace-pre-wrap break-words leading-relaxed">
                  {lastCoralQuery
                    .split(
                      /(\bSELECT\b|\bFROM\b|\bWHERE\b|\bAND\b|\bLIMIT\b|'.*?')/g,
                    )
                    .map((part, i) => {
                      if (
                        ["SELECT", "FROM", "WHERE", "AND", "LIMIT"].includes(
                          part,
                        )
                      )
                        return (
                          <span
                            key={i}
                            className="text-[#2383e2] font-semibold"
                          >
                            {part}
                          </span>
                        );
                      else if (part.startsWith("'") && part.endsWith("'"))
                        return (
                          <span key={i} className="text-[#eb5757]">
                            {part}
                          </span>
                        );
                      return <span key={i}>{part}</span>;
                    })}
                </pre>
              </div>
            ) : (
              <div className="text-[#787774] flex-grow flex items-center italic">
                Awaiting telemetry query...
              </div>
            )}
          </div>
        </div>

        {/* RECENT PRS */}
        <div>
          <h2 className="text-base font-semibold text-[#37352f] mb-3 border-b border-[#e9e9e7] pb-2 flex items-center gap-2">
            <GitPullRequest
              className="w-4 h-4 text-[#37352f]"
              strokeWidth={1.5}
            />{" "}
            Generated Pull Requests
          </h2>
          <div className="min-h-[120px] flex flex-col">
            {recentPRs.length === 0 ? (
              <div className="py-4 text-[13px] text-[#787774] flex-grow flex items-center">
                No PRs generated yet.
              </div>
            ) : (
              <div className="divide-y divide-[#e9e9e7]">
                {recentPRs.map((inc) => (
                  <div
                    key={inc.id}
                    className="flex items-center justify-between py-3 hover:bg-[#fbfbfa] transition-colors group -mx-2 px-2 rounded-md"
                  >
                    <div className="flex items-center gap-3">
                      <GitPullRequest
                        className="w-4 h-4 text-[#787774]"
                        strokeWidth={1.5}
                      />
                      <div>
                        <p className="text-[13px] font-medium text-[#37352f]">
                          engram/fix-{inc.head_sha?.substring(0, 7)}
                        </p>
                        <p className="text-[12px] text-[#787774]">{inc.repo}</p>
                      </div>
                    </div>
                    <a
                      href={inc.pr_url}
                      target="_blank"
                      rel="noreferrer"
                      className="text-[12px] text-[#787774] hover:text-[#37352f] flex items-center gap-1 underline decoration-[#e9e9e7] hover:decoration-[#37352f] underline-offset-2 transition-all"
                    >
                      Review{" "}
                      <ExternalLink className="w-3 h-3" strokeWidth={1.5} />
                    </a>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  icon,
}: {
  title: string;
  value: string | number;
  icon: React.ReactNode;
}) {
  return (
    <div className="border border-[#e9e9e7] rounded-lg p-4 flex flex-col justify-between group hover:bg-[#fbfbfa] transition-colors">
      <div className="flex items-start justify-between mb-1">
        <h3 className="text-[13px] font-medium text-[#787774]">{title}</h3>
        {icon}
      </div>
      <span className="text-[28px] tracking-tight font-semibold text-[#37352f]">
        {value}
      </span>
    </div>
  );
}

function StatusBadge({ stage }: { stage: string }) {
  if (stage === "healed")
    return (
      <span className="inline-flex items-center px-1.5 py-0.5 rounded-[4px] text-[12px] font-medium bg-[#e6f4ea] text-[#1e4620]">
        Healed
      </span>
    );
  if (stage === "failed")
    return (
      <span className="inline-flex items-center px-1.5 py-0.5 rounded-[4px] text-[12px] font-medium bg-[#fce8e6] text-[#c1292e]">
        Failed
      </span>
    );
  if (stage === "pending_approval")
    return (
      <span className="inline-flex items-center px-1.5 py-0.5 rounded-[4px] text-[12px] font-medium bg-[#fdf3c0] text-[#6b4a0d]">
        Needs Auth
      </span>
    );
  return (
    <span className="inline-flex items-center gap-1.5 px-1.5 py-0.5 rounded-[4px] text-[12px] font-medium bg-[#f1f1ef] text-[#37352f]">
      <div className="w-1.5 h-1.5 bg-[#787774] rounded-full animate-pulse" />
      {stage === "diagnosing"
        ? "Diagnosing"
        : stage === "aggregating"
          ? "Aggregating"
          : "Sandboxing"}
    </span>
  );
}
