"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Eye,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  MoreHorizontal,
  Siren,
  MessageSquare,
} from "lucide-react";

export default function ClientLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [online, setOnline] = useState(false);
  const pathname = usePathname();

  useEffect(() => {
    const check = async () => {
      try {
        setOnline((await fetch("http://127.0.0.1:8080/health")).ok);
      } catch {
        setOnline(false);
      }
    };
    check();
    const iv = setInterval(check, 3000);
    return () => clearInterval(iv);
  }, []);

  const navItems = [
    { name: "Mission Control", href: "/", icon: LayoutDashboard },
    { name: "Chat", href: "/chat", icon: MessageSquare },
    { name: "Observability", href: "/observability", icon: Eye },
    { name: "War Room", href: "/war-room", icon: Siren },
    { name: "Settings", href: "/settings", icon: Settings },
  ];

  const currentPage =
    navItems.find((i) => i.href === pathname)?.name ?? "Engram";

  return (
    <>
      <style>{`
        /* ── Layout shell ── */
        .layout-root {
          display: flex;
          height: 100vh;
          width: 100%;
          background: #ffffff;
          color: #37352f;
          font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
          overflow: hidden;
        }

        /* ── Sidebar ── */
        .layout-sidebar {
          display: flex;
          flex-direction: column;
          background: #f7f7f5;
          border-right: 1px solid #e9e9e7;
          flex-shrink: 0;
          transition: width 0.22s cubic-bezier(0.4,0,0.2,1);
          overflow: hidden;
        }

        .layout-sidebar.open  { width: 236px; }
        .layout-sidebar.closed { width: 0; border-right: none; }

        .layout-sidebar-inner { width: 236px; display: flex; flex-direction: column; height: 100%; }

        /* Workspace header */
        .layout-ws-hdr {
          height: 44px;
          display: flex;
          align-items: center;
          padding: 0 10px;
          margin-top: 4px;
        }

        .layout-ws-btn {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 5px 8px;
          border-radius: 5px;
          cursor: pointer;
          width: 100%;
          transition: background 0.1s;
          border: none;
          background: none;
          color: #37352f;
          font-size: 14px;
          font-weight: 500;
          font-family: inherit;
        }
        .layout-ws-btn:hover { background: #efefed; }

        .layout-ws-logo {
          width: 22px;
          height: 22px;
          background: #37352f;
          border-radius: 4px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          font-size: 11px;
          font-weight: 800;
          flex-shrink: 0;
          letter-spacing: -0.02em;
        }

        /* Nav */
        .layout-nav { padding: 4px 8px; display: flex; flex-direction: column; gap: 1px; }

        .layout-nav-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 6px 10px;
          border-radius: 5px;
          font-size: 13.5px;
          font-weight: 500;
          color: #787774;
          text-decoration: none;
          transition: background 0.1s, color 0.1s;
          white-space: nowrap;
        }

        .layout-nav-item:hover          { background: #efefed; color: #37352f; }
        .layout-nav-item.active         { background: #efefed; color: #37352f; }
        .layout-nav-item.active svg     { color: #37352f; }

        /* Section label */
        .layout-section-label {
          padding: 16px 18px 4px;
          font-size: 11px;
          font-weight: 600;
          color: #aeaca8;
          letter-spacing: 0.04em;
        }

        .layout-deploy-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 6px 18px;
          font-size: 13px;
          color: #787774;
          cursor: default;
        }

        .layout-deploy-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: #0f7b6c;
          flex-shrink: 0;
        }

        /* ── Main area ── */
        .layout-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }

        /* Topbar */
        .layout-topbar {
          height: 44px;
          display: flex;
          align-items: center;
          padding: 0 16px;
          flex-shrink: 0;
          border-bottom: 1px solid transparent;
          background: rgba(255,255,255,0.85);
          backdrop-filter: blur(8px);
          -webkit-backdrop-filter: blur(8px);
          position: sticky;
          top: 0;
          z-index: 10;
          gap: 6px;
        }

        .layout-topbar-toggle {
          padding: 4px;
          border-radius: 4px;
          border: none;
          background: none;
          cursor: pointer;
          color: #787774;
          display: flex;
          transition: background 0.1s, color 0.1s;
        }
        .layout-topbar-toggle:hover { background: #efefed; color: #37352f; }

        .layout-breadcrumb {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 13px;
          color: #aeaca8;
        }

        .layout-breadcrumb-sep { color: #d9d9d6; }

        .layout-breadcrumb-current {
          color: #37352f;
          font-weight: 500;
        }

        /* Status pill */
        .layout-status {
          margin-left: auto;
          display: flex;
          align-items: center;
          gap: 5px;
          font-size: 11px;
          font-weight: 500;
          color: #787774;
          padding: 3px 10px;
          border-radius: 20px;
          border: 1px solid #e9e9e7;
          background: #fafafa;
        }

        .layout-status-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          transition: background 0.3s;
        }
        .layout-status-dot.on  { background: #0f7b6c; }
        .layout-status-dot.off { background: #eb5757; }

        .layout-more-btn {
          margin-left: 4px;
          padding: 4px;
          border-radius: 4px;
          border: none;
          background: none;
          cursor: pointer;
          color: #787774;
          display: flex;
          transition: background 0.1s;
        }
        .layout-more-btn:hover { background: #efefed; }

        /* Canvas */
        .layout-canvas { flex: 1; overflow-y: auto; }
      `}</style>

      <div className="layout-root">
        {/* Sidebar */}
        <aside className={`layout-sidebar ${sidebarOpen ? "open" : "closed"}`}>
          <div className="layout-sidebar-inner">
            {/* Workspace name */}
            <div className="layout-ws-hdr">
              <button className="layout-ws-btn">
                <div className="layout-ws-logo">E</div>
                <span>Engram Workspace</span>
              </button>
            </div>

            {/* Navigation */}
            <nav className="layout-nav">
              {navItems.map((item) => (
                <Link
                  key={item.name}
                  href={item.href}
                  className={`layout-nav-item ${pathname === item.href ? "active" : ""}`}
                >
                  <item.icon size={15} strokeWidth={1.8} />
                  {item.name}
                </Link>
              ))}
            </nav>

            {/* Active deployments */}
            <div className="layout-section-label">Active Deployments</div>
            <div className="layout-deploy-item">
              <div className="layout-deploy-dot" />
              engram-test-repo
            </div>
          </div>
        </aside>

        {/* Main */}
        <div className="layout-main">
          {/* Topbar */}
          <header className="layout-topbar">
            <button
              className="layout-topbar-toggle"
              onClick={() => setSidebarOpen(!sidebarOpen)}
              title={sidebarOpen ? "Close sidebar" : "Open sidebar"}
            >
              {sidebarOpen ? (
                <PanelLeftClose size={15} strokeWidth={1.5} />
              ) : (
                <PanelLeftOpen size={15} strokeWidth={1.5} />
              )}
            </button>

            <div className="layout-breadcrumb">
              <span style={{ cursor: "pointer" }} className="hidden sm:inline">
                Engram Workspace
              </span>
              <span className="layout-breadcrumb-sep hidden sm:inline">/</span>
              <span className="layout-breadcrumb-current">{currentPage}</span>
            </div>

            <div className="layout-status">
              <div className={`layout-status-dot ${online ? "on" : "off"}`} />
              {online ? "Connected" : "Offline"}
            </div>

            <button className="layout-more-btn">
              <MoreHorizontal size={15} />
            </button>
          </header>

          {/* Page canvas */}
          <main className="layout-canvas">{children}</main>
        </div>
      </div>
    </>
  );
}
