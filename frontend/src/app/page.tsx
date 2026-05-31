"use client";
import React, { useState } from "react";
import Link from "next/link";
import {
  Terminal,
  GitPullRequest,
  Database,
  CheckCircle2,
  Cpu,
  Mic,
  Activity,
  ArrowRight,
  Layers,
  Server,
  Code,
  Sparkles,
  ChevronRight,
} from "lucide-react";

export default function EngramLandingPage() {
  const [activeSqlTab, setActiveSqlTab] = useState("commits");

  const sqlSnippets = {
    commits: `-- Find which commit triggered a failure\nSELECT sha, author__login, commit__message \nFROM github.commits\nWHERE owner = 'gojogourav' AND repo = 'engram-test-repo'\nLIMIT 10;`,
    workflows: `-- Get recent workflow run outcomes\nSELECT id, workflow_id, status, conclusion, head_sha\nFROM github.repo_action_workflow_runs\nWHERE owner = 'gojogourav' AND repo = 'engram-test-repo'\nAND conclusion = 'failure'\nLIMIT 5;`,
    jobs: `-- Find which job step failed\nSELECT name, conclusion, failed_step_names\nFROM github.jobs\nWHERE owner = 'gojogourav' AND repo = 'engram-test-repo'\nAND run_id = 987654321;`,
  };

  return (
    <div className="min-h-screen bg-white dark:bg-[#191919] text-zinc-800 dark:text-zinc-200 font-sans selection:bg-zinc-200/80 dark:selection:bg-zinc-800">
      {/* Top Navigation */}
      <nav className="border-b border-zinc-200 dark:border-zinc-800 sticky top-0 bg-white/80 dark:bg-[#191919]/80 backdrop-blur-md z-50">
        <div className="max-w-5xl mx-auto px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-5 h-5 rounded bg-zinc-900 dark:bg-zinc-100 flex items-center justify-center">
              <span className="text-white dark:text-zinc-900 text-xs font-bold font-mono">
                E
              </span>
            </div>
            <span className="font-semibold tracking-tight text-zinc-900 dark:text-zinc-50 text-sm">
              Engram
            </span>
            <span className="text-[10px] font-mono bg-zinc-100 dark:bg-zinc-800 text-zinc-500 dark:text-zinc-400 px-1.5 py-0.5 rounded border border-zinc-200 dark:border-zinc-700/60">
              v1.0.0
            </span>
          </div>
          <div className="flex items-center gap-6 text-xs font-medium text-zinc-500 dark:text-zinc-400">
            <a
              href="#features"
              className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors hidden sm:block"
            >
              Features
            </a>
            <a
              href="#coral"
              className="hover:text-zinc-900 dark:hover:text-zinc-100 transition-colors hidden sm:block"
            >
              Coral Data Fabric
            </a>
            <Link
              href="/home"
              className="bg-zinc-900 hover:bg-zinc-800 dark:bg-zinc-100 dark:hover:bg-zinc-200 text-white dark:text-zinc-900 px-4 py-1.5 rounded transition-colors shadow-sm font-semibold flex items-center gap-1.5"
            >
              Launch <ArrowRight className="w-3 h-3" />
            </Link>
          </div>
        </div>
      </nav>

      {/* Main Content Container (No Sidebar) */}
      <main className="max-w-4xl mx-auto px-6 py-16 space-y-20">
        {/* Hero Section */}
        <header className="space-y-6 max-w-2xl">
          <div className="inline-flex items-center gap-2 text-xs font-mono text-zinc-400 dark:text-zinc-500 border border-zinc-200 dark:border-zinc-800 px-2.5 py-1 rounded-full bg-zinc-50/50 dark:bg-zinc-950/20">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
            Autonomous SRE Agent Active
          </div>
          <h1 className="text-4xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50 sm:text-5xl">
            CI breaks. Engram fixes it. <br />
            <span className="text-zinc-400 dark:text-zinc-500">
              Automatically.
            </span>
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400 leading-relaxed max-w-xl">
            An AI-powered Site Reliability Engineering engine that intercepts
            production & deployment pipeline disruptions, conducts direct
            multi-modal root-cause analysis using Gemini 2.5 Flash, formats
            unified patches, and provisions containment pull-requests without
            structural engineering delays.
          </p>

          <div className="flex items-center gap-4 pt-2">
            <Link
              href="/home"
              className="bg-zinc-900 hover:bg-zinc-800 dark:bg-zinc-100 dark:hover:bg-zinc-200 text-white dark:text-zinc-900 px-5 py-2.5 rounded-lg text-sm font-semibold transition-colors shadow-sm flex items-center gap-2"
            >
              Launch Platform <ArrowRight className="w-4 h-4" />
            </Link>
            <a
              href="#docs"
              className="text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100 text-sm font-medium transition-colors px-4 py-2.5"
            >
              Documentation
            </a>
          </div>
        </header>

        {/* Notion Style Callout: Live Value Proposition */}
        <div className="bg-zinc-50/60 dark:bg-[#202020]/30 border border-zinc-200 dark:border-zinc-800 rounded-xl p-5 flex gap-4 items-start">
          <div className="p-2 bg-zinc-200/50 dark:bg-zinc-800/60 rounded-md text-zinc-700 dark:text-zinc-300 flex-shrink-0">
            <Sparkles className="w-4 h-4" />
          </div>
          <div className="space-y-1.5">
            <h4 className="text-xs font-semibold text-zinc-900 dark:text-zinc-100 uppercase tracking-wider">
              Operational Overview
            </h4>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
              Engram bridges infrastructure silos by embedding a multi-step
              natural language agent with complete contextual authorization
              across Kubernetes container states and the internal **Coral
              Federated SQL engine**.
            </p>
          </div>
        </div>

        <hr className="border-zinc-200 dark:border-zinc-800" />

        {/* Section: Autonomous Healing Loop */}
        <section className="space-y-6">
          <div className="space-y-1">
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
              <Activity className="w-4 h-4 text-zinc-400" /> The Autonomous
              Self-Healing Pipeline
            </h2>
            <p className="text-xs text-zinc-400 dark:text-zinc-500">
              Traced infrastructure state engine from standard webhook failures
              down to applied codebase adjustments.
            </p>
          </div>

          {/* Minimal Lifecycle Flow Map */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3 font-mono text-[11px]">
            {[
              {
                step: "01 / INGESTION",
                title: "GitHub Webhook Fails",
                desc: "Pipeline extracts broken steps and downloads full context logs automatically.",
              },
              {
                step: "02 / FEDERATION",
                title: "Coral Context Pull",
                desc: "Gathers matching commits, environment metadata, and telemetry without API clients.",
              },
              {
                step: "03 / ANALYSIS",
                title: "Gemini LogParser",
                desc: "Translates stack traces, determines root errors, and constructs custom file modifications.",
              },
              {
                step: "04 / DELIVERY",
                title: "Approval Sandbox",
                desc: "Applies local diff tests, triggers a human gate threshold, and generates verified branch PRs.",
              },
            ].map((node, i) => (
              <div
                key={i}
                className="border border-zinc-200 dark:border-zinc-800 rounded-lg p-3.5 bg-zinc-50/30 dark:bg-[#1a1a1a]/40 space-y-2"
              >
                <div className="text-zinc-400 dark:text-zinc-500 font-semibold tracking-wider text-[10px]">
                  {node.step}
                </div>
                <div className="font-semibold text-zinc-900 dark:text-zinc-100 flex items-center gap-1.5">
                  {i === 3 ? (
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" />
                  ) : (
                    <ChevronRight className="w-3 h-3 text-zinc-400" />
                  )}
                  {node.title}
                </div>
                <p className="text-zinc-500 dark:text-zinc-400 leading-normal font-sans text-xs">
                  {node.desc}
                </p>
              </div>
            ))}
          </div>
        </section>

        {/* Section: Core Features Grid */}
        <section id="features" className="space-y-6 pt-4">
          <div className="space-y-1">
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
              <Layers className="w-4 h-4 text-zinc-400" /> System Features
            </h2>
            <p className="text-xs text-zinc-400 dark:text-zinc-500">
              Enterprise automation primitives designed around safety boundaries
              and developer velocity.
            </p>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {[
              {
                icon: <Terminal className="w-4 h-4 text-zinc-500" />,
                title: "ChatOps Command Subsystems",
                desc: "Run unified commands natively using text interfaces. Pass configurations to /k8s, /docker, or /grafana routes to verify live cluster topologies instantaneously.",
              },
              {
                icon: <GitPullRequest className="w-4 h-4 text-zinc-500" />,
                title: "10-Min Human Approval Gates",
                desc: "All generated file patches require explicit interactive consent. Halts at execution gates until authorized via ChatOps or the integrated Mission Control War Room UI.",
              },
              {
                icon: <Cpu className="w-4 h-4 text-zinc-500" />,
                title: "Prometheus Self-Correction",
                desc: "Bypasses standard logging constraints by intercepting Alertmanager webhooks. Automatically matches OOMKilled or CrashLoopBackOff anomalies to safe scale configurations.",
              },
              {
                icon: <Mic className="w-4 h-4 text-zinc-500" />,
                title: "Hands-Free Voice Transcription",
                desc: "Incorporates low-latency Groq Whisper large translation modules. Stream clear base64 audio directly to instantly coordinate high-velocity triage workflows.",
              },
            ].map((feat, i) => (
              <div
                key={i}
                className="border border-zinc-200 dark:border-zinc-800 rounded-xl p-4 space-y-2 hover:bg-zinc-50/50 dark:hover:bg-[#202020]/20 transition-colors"
              >
                <div className="flex items-center gap-2">
                  <div className="p-1 bg-zinc-100 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700/50 rounded">
                    {feat.icon}
                  </div>
                  <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
                    {feat.title}
                  </h3>
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed pl-8">
                  {feat.desc}
                </p>
              </div>
            ))}
          </div>
        </section>

        {/* Section: The Coral Federated Engine */}
        <section id="coral" className="space-y-6 pt-4">
          <div className="space-y-2 max-w-xl">
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
              <Database className="w-4 h-4 text-zinc-400" /> Coral Federated
              Logic Fabric
            </h2>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
              Coral unifies diverse metrics platforms, GitHub log repositories,
              and infrastructure schemas into standardized database instances.
              It removes manual handling of custom rate limiters and multi-stage
              token handshakes.
            </p>
          </div>

          {/* Comparison Grid Table */}
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-xl overflow-hidden bg-white dark:bg-[#1c1c1c]">
            <div className="grid grid-cols-2 bg-zinc-50/60 dark:bg-[#202020]/40 text-xs font-semibold text-zinc-500 dark:text-zinc-400 border-b border-zinc-200 dark:border-zinc-800 divide-x divide-zinc-200 dark:divide-zinc-800">
              <div className="p-3">TRADITIONAL CLIENT INTEGRATION</div>
              <div className="p-3 text-zinc-800 dark:text-zinc-200 flex items-center gap-1.5">
                <Sparkles className="w-3.5 h-3.5 text-blue-500" /> VIA CORAL
                QUERY FABRIC
              </div>
            </div>
            <div className="divide-y divide-zinc-200 dark:divide-zinc-800 text-xs text-zinc-600 dark:text-zinc-400 font-medium">
              {[
                {
                  old: "3 distinct API client functions to map individual target commits",
                  new: "Single atomic cross-table relational selection query",
                },
                {
                  old: "Manual curation of token handshakes and response object structures",
                  new: "Centralized credential layer handled by deep unified schema mapping",
                },
                {
                  old: "Complex programmatic tracking of pagination metadata loops",
                  new: "Simple Declarative SQL adjustments utilizing basic LIMIT directives",
                },
                {
                  old: "50+ lines of custom Go architectural boilerplate client logic",
                  new: "A simple 1-line database invocation string parameter",
                },
              ].map((row, idx) => (
                <div
                  key={idx}
                  className="grid grid-cols-2 divide-x divide-zinc-200 dark:divide-zinc-800 hover:bg-zinc-50/30 dark:hover:bg-zinc-800/10 transition-colors"
                >
                  <div className="p-3 text-zinc-400 dark:text-zinc-500 font-mono text-[11px]">
                    {row.old}
                  </div>
                  <div className="p-3 font-sans text-zinc-800 dark:text-zinc-300 pl-4">
                    {row.new}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Interactive SQL Sample Viewer */}
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-xl overflow-hidden bg-[#1c1c1c] text-zinc-300">
            <div className="flex border-b border-zinc-800 bg-[#161616] px-4 py-1.5 gap-2">
              {Object.keys(sqlSnippets).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setActiveSqlTab(tab)}
                  className={`text-[11px] font-mono px-2.5 py-1 rounded transition-colors capitalize ${
                    activeSqlTab === tab
                      ? "bg-zinc-800 text-zinc-100 font-semibold"
                      : "text-zinc-500 hover:text-zinc-300"
                  }`}
                >
                  {tab}.sql
                </button>
              ))}
            </div>
            <div className="p-4 bg-[#191919] overflow-x-auto">
              <pre className="font-mono text-xs leading-relaxed text-emerald-400/90 selection:bg-zinc-700">
                {sqlSnippets[activeSqlTab as keyof typeof sqlSnippets]}
              </pre>
            </div>
          </div>
        </section>

        {/* Section: Architecture Stack Mapping */}
        <section id="architecture" className="space-y-6 pt-4">
          <div className="space-y-1">
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
              <Server className="w-4 h-4 text-zinc-400" /> Architectural
              Composition
            </h2>
            <p className="text-xs text-zinc-400 dark:text-zinc-500">
              System dependencies mapping components cleanly from backend
              runtime environments down to core model execution targets.
            </p>
          </div>

          <div className="border border-zinc-200 dark:border-zinc-800 rounded-xl overflow-hidden">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-zinc-50 dark:bg-[#202020]/40 text-zinc-400 dark:text-zinc-500 border-b border-zinc-200 dark:border-zinc-800 font-semibold font-mono uppercase tracking-wider text-[10px]">
                  <th className="p-3 pl-4">Platform Layer</th>
                  <th className="p-3">Integrated Technology Target</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800 font-medium text-zinc-700 dark:text-zinc-300">
                {[
                  {
                    layer: "Core Orchestration Backend",
                    tech: "Go 1.22 Runtime System Engine",
                  },
                  {
                    layer: "AI Reasoning Model",
                    tech: "Gemini 2.5 Flash Framework Integration",
                  },
                  {
                    layer: "Data Abstraction Interface",
                    tech: "Coral Unified Federated SQL Engine Client",
                  },
                  {
                    layer: "Cluster Management Client",
                    tech: "Kubernetes Custom Client-Go Bindings",
                  },
                  {
                    layer: "Telemetry Analytics Tracking",
                    tech: "Prometheus Monitoring Scraping System & promhttp",
                  },
                  {
                    layer: "Interactive User Front-end",
                    tech: "Next.js 14 Framework Layouts & Tailwind CSS",
                  },
                ].map((item, idx) => (
                  <tr
                    key={idx}
                    className="hover:bg-zinc-50/40 dark:hover:bg-zinc-800/10 transition-colors"
                  >
                    <td className="p-3 pl-4 font-mono text-[11px] text-zinc-400 dark:text-zinc-500">
                      {item.layer}
                    </td>
                    <td className="p-3 font-semibold text-zinc-800 dark:text-zinc-200">
                      {item.tech}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Section: Getting Started */}
        <section id="docs" className="space-y-4 pt-4 pb-12">
          <div className="space-y-1">
            <h2 className="text-lg font-bold text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
              <Code className="w-4 h-4 text-zinc-400" /> Initial Platform
              Bootstrap
            </h2>
            <p className="text-xs text-zinc-400 dark:text-zinc-500">
              Initialize runtime servers locally along with tracking webhooks
              configuration profiles.
            </p>
          </div>

          <div className="space-y-3 font-mono text-xs">
            <div className="space-y-1.5">
              <span className="text-zinc-400 text-[11px] font-semibold block">
                1. Fire up the Orchestration Backend Daemon
              </span>
              <div className="bg-zinc-50 dark:bg-[#202020]/30 border border-zinc-200 dark:border-zinc-800 text-zinc-700 dark:text-zinc-300 rounded-lg p-3">
                cd cmd && go run main.go
              </div>
            </div>

            <div className="space-y-1.5">
              <span className="text-zinc-400 text-[11px] font-semibold block">
                2. Launch Next.js Mission Control Frontend UI
              </span>
              <div className="bg-zinc-50 dark:bg-[#202020]/30 border border-zinc-200 dark:border-zinc-800 text-zinc-700 dark:text-zinc-300 rounded-lg p-3">
                cd frontend && npm install && npm run dev
              </div>
            </div>
          </div>

          <div className="bg-zinc-50/50 dark:bg-[#202020]/10 border border-zinc-200 dark:border-zinc-800 rounded-lg p-4 text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed font-sans">
            <strong>Automated Routing Notice:</strong> Upon local configuration
            initialisation, Engram establishes encrypted, isolated proxy tunnels
            using the embedded ngrok SDK instances, pairing live event handlers
            cleanly onto target webhook destinations instantly.
          </div>
        </section>
      </main>

      {/* Modern Minimal Footer */}
      <footer className="border-t border-zinc-200 dark:border-zinc-800 py-8 bg-zinc-50/30 dark:bg-[#1a1a1a]/20">
        <div className="max-w-4xl mx-auto px-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs font-mono text-zinc-400 dark:text-zinc-500">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-zinc-700 dark:text-zinc-300 font-sans text-sm">
              Engram
            </span>
            <span>·</span>
            <span>Autonomous SRE Protocol Framework</span>
          </div>
          <div className="flex gap-4">
            <a
              href="#features"
              className="hover:text-zinc-700 dark:hover:text-zinc-300 transition-colors"
            >
              Features
            </a>
            <a
              href="#coral"
              className="hover:text-zinc-700 dark:hover:text-zinc-300 transition-colors"
            >
              Coral Engine
            </a>
            <a
              href="#architecture"
              className="hover:text-zinc-700 dark:hover:text-zinc-300 transition-colors"
            >
              Architecture
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
