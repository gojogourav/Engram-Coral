"use client";

import { useState } from "react";
import {
  Settings,
  Save,
  // Github,
  Key,
  Server,
  LineChart,
  CheckCircle2,
  Shield,
} from "lucide-react";

export default function SettingsPage() {
  const [formData, setFormData] = useState({
    repo: "yujiblack/Vortex-Test",
    token: "",
    secret: "",
    kubeconfig: "",
    grafanaUrl: "http://localhost:9090",
  });

  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState("");

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setIsSuccess(false);
    setError("");

    if (!formData.repo || !formData.token || !formData.secret) {
      setError("Repository, GitHub Token, and Webhook Secret are required.");
      setIsLoading(false);
      return;
    }

    try {
      console.log("Attempting to connect to backend...");

      const res = await fetch("http://localhost:8080/repos/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          repo: formData.repo,
          github_token: formData.token,
          webhook_secret: formData.secret,
          kubeconfig_b64: formData.kubeconfig,
          grafana_url: formData.grafanaUrl,
        }),
      });

      if (!res.ok) {
        // Grab the exact error message from the Go backend
        const errText = await res.text();
        throw new Error(`Backend rejected (Status ${res.status}): ${errText}`);
      }

      setIsSuccess(true);
      setTimeout(() => setIsSuccess(false), 3000);
    } catch (err: any) {
      console.error("Full error object:", err);
      // Display the actual error on the screen
      setError(err.message || "An unknown network error occurred");
    } finally {
      setIsLoading(false);
    }
  };
  return (
    <main className="max-w-6xl mx-auto px-4 sm:px-6 mt-6 sm:mt-10 pb-20 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <header className="mb-8">
        <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-[#37352f] mb-2 flex items-center gap-3">
          <Settings className="w-8 h-8 text-[#37352f]" />
          Fleet Integration
        </h1>
        <p className="text-[#6b6b6b] text-sm sm:text-base">
          Bind your repositories and infrastructure to the Engram Engine.
        </p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* LEFT COLUMN: THE FORM */}
        <div className="lg:col-span-2 space-y-6">
          {isSuccess && (
            <div className="bg-[#e6f6f1] border border-[#b5e3d8] rounded-xl p-4 flex items-center gap-3 animate-in slide-in-from-top-2">
              <CheckCircle2 className="w-5 h-5 text-[#0f7b6c]" />
              <p className="text-sm font-medium text-[#0f7b6c]">
                Repository successfully registered and bound to pipeline.
              </p>
            </div>
          )}

          {error && (
            <div className="bg-[#fff5f5] border border-[#ffd4d4] rounded-xl p-4 flex items-center gap-3 animate-in slide-in-from-top-2">
              <Shield className="w-5 h-5 text-[#e03e3e]" />
              <p className="text-sm font-medium text-[#e03e3e]">{error}</p>
            </div>
          )}

          <form
            onSubmit={handleSave}
            className="bg-white border border-[#e9e9e7] rounded-xl shadow-sm overflow-hidden"
          >
            <div className="p-6 space-y-6">
              {/* GitHub Section */}
              <div>
                <h3 className="text-sm font-bold text-[#9b9a97] uppercase tracking-wider mb-4 flex items-center gap-2">
                  {/*<Github className="w-4 h-4 text-[#37352f]" /> GitOps*/}
                  Github Configuration
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-[#6b6b6b] mb-1">
                      Target Repository
                    </label>
                    <input
                      type="text"
                      value={formData.repo}
                      onChange={(e) =>
                        setFormData({ ...formData, repo: e.target.value })
                      }
                      placeholder="owner/repo"
                      className="w-full px-3 py-2 bg-[#f7f7f5] border border-[#e9e9e7] rounded-md text-sm outline-none focus:border-[#37352f] transition-colors"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-[#6b6b6b] mb-1">
                      GitHub Personal Access Token
                    </label>
                    <input
                      type="password"
                      value={formData.token}
                      onChange={(e) =>
                        setFormData({ ...formData, token: e.target.value })
                      }
                      placeholder="ghp_xxxxxxxxxxxx"
                      className="w-full px-3 py-2 bg-[#f7f7f5] border border-[#e9e9e7] rounded-md text-sm outline-none focus:border-[#37352f] transition-colors"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-[#6b6b6b] mb-1">
                      Webhook Secret (For validation)
                    </label>
                    <input
                      type="password"
                      value={formData.secret}
                      onChange={(e) =>
                        setFormData({ ...formData, secret: e.target.value })
                      }
                      placeholder="Your secure secret"
                      className="w-full px-3 py-2 bg-[#f7f7f5] border border-[#e9e9e7] rounded-md text-sm outline-none focus:border-[#37352f] transition-colors"
                    />
                  </div>
                </div>
              </div>

              <div className="h-px bg-[#e9e9e7] w-full" />

              {/* Infrastructure Section */}
              <div>
                <h3 className="text-sm font-bold text-[#9b9a97] uppercase tracking-wider mb-4 flex items-center gap-2">
                  <Server className="w-4 h-4 text-[#2383e2]" /> Infrastructure
                  Binding
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-[#6b6b6b] mb-1">
                      Kubeconfig (Base64 Encoded)
                    </label>
                    <textarea
                      value={formData.kubeconfig}
                      onChange={(e) =>
                        setFormData({ ...formData, kubeconfig: e.target.value })
                      }
                      placeholder="LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS..."
                      rows={3}
                      className="w-full px-3 py-2 bg-[#f7f7f5] border border-[#e9e9e7] rounded-md text-sm outline-none focus:border-[#37352f] transition-colors resize-none font-mono"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-[#6b6b6b] mb-1 flex items-center gap-1">
                      <LineChart className="w-3 h-3 text-[#dfab01]" /> Grafana
                      Instance URL
                    </label>
                    <input
                      type="text"
                      value={formData.grafanaUrl}
                      onChange={(e) =>
                        setFormData({ ...formData, grafanaUrl: e.target.value })
                      }
                      placeholder="http://localhost:9090"
                      className="w-full px-3 py-2 bg-[#f7f7f5] border border-[#e9e9e7] rounded-md text-sm outline-none focus:border-[#37352f] transition-colors"
                    />
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-[#f7f7f5] px-6 py-4 border-t border-[#e9e9e7] flex justify-end">
              <button
                type="submit"
                disabled={isLoading}
                className="bg-[#37352f] hover:bg-[#2f2d28] disabled:bg-[#9b9a97] text-white px-6 py-2 rounded-md text-sm font-medium transition-colors flex items-center gap-2"
              >
                {isLoading ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>{" "}
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="w-4 h-4" /> Save Configuration
                  </>
                )}
              </button>
            </div>
          </form>
        </div>

        {/* RIGHT COLUMN: ABOUT */}
        <div className="space-y-6">
          <div className="bg-white border border-[#e9e9e7] rounded-xl p-6 shadow-sm">
            <h3 className="text-xs font-bold text-[#9b9a97] uppercase tracking-wider mb-4 flex items-center gap-2">
              <Key className="w-4 h-4 text-[#9065b0]" /> Security Notice
            </h3>
            <p className="text-sm text-[#6b6b6b] leading-relaxed mb-4">
              Engram requires elevated permissions to automatically generate
              pull requests and mutate cluster state.
            </p>
            <p className="text-sm text-[#6b6b6b] leading-relaxed">
              Your tokens are processed locally by the Go binary and are never
              transmitted to external servers other than GitHub and your
              cluster.
            </p>
          </div>

          <div className="bg-[#fcfcfc] border border-[#e9e9e7] rounded-xl p-6 shadow-sm">
            <h3 className="text-sm font-semibold text-[#37352f] mb-3">
              System Specifications
            </h3>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-[#9b9a97]">Architecture</span>
                <span className="font-medium text-[#37352f]">Go + Next.js</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#9b9a97]">AI Engine</span>
                <span className="font-medium text-[#37352f]">
                  Gemini 2.5 Flash
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#9b9a97]">Data Fabric</span>
                <span className="font-medium text-[#37352f]">Coral SQL</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#9b9a97]">Version</span>
                <span className="font-medium text-[#37352f]">
                  v1.0.0-hackathon
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
