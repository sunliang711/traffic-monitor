import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { del, get, patch, post, put } from "./api";
import type {
  AdminProfile,
  AlertItem,
  AlertList,
  CollectNowResponse,
  ConnectionTestResponse,
  Machine,
  NotificationChannel,
  SSHKey,
  WebhookTestResponse,
  ThresholdRule,
  TrafficSample,
  TrafficSampleList,
} from "./types";

type TabKey =
  | "overview"
  | "machines"
  | "sshKeys"
  | "thresholds"
  | "notifications"
  | "samples"
  | "alerts";

type LoginFormState = {
  username: string;
  password: string;
};

type SSHKeyImportState = {
  name: string;
  privateKey: string;
};

type SSHKeyGenerateState = {
  name: string;
};

type MachineFormState = {
  name: string;
  host: string;
  port: string;
  sshUser: string;
  networkInterface: string;
  sshKeyID: string;
  collectEnabled: boolean;
  remark: string;
};

type ThresholdFormRow = {
  period_type: string;
  metric_type: string;
  threshold_value: string;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
  source?: string;
};

type WebhookFormState = {
  enabled: boolean;
  method: "GET" | "POST";
  url: string;
  headersText: string;
  bodyText: string;
};

type WebhookPreviewState = {
  url: string;
  headersText: string;
  bodyText: string;
};



type TelegramFormState = {
  enabled: boolean;
  botToken: string;
  chatID: string;
};

const tabs: Array<{ key: TabKey; label: string }> = [
  { key: "overview", label: "总览" },
  { key: "machines", label: "机器" },
  { key: "sshKeys", label: "SSH Key" },
  { key: "thresholds", label: "阈值" },
  { key: "notifications", label: "通知" },
  { key: "samples", label: "样本" },
  { key: "alerts", label: "告警" },
];

const thresholdDimensions = [
  { period_type: "hourly", metric_type: "upload" },
  { period_type: "hourly", metric_type: "download" },
  { period_type: "hourly", metric_type: "total" },
  { period_type: "daily", metric_type: "upload" },
  { period_type: "daily", metric_type: "download" },
  { period_type: "daily", metric_type: "total" },
] as const;

const emptyMachineForm = (): MachineFormState => ({
  name: "",
  host: "",
  port: "22",
  sshUser: "",
  networkInterface: "",
  sshKeyID: "",
  collectEnabled: true,
  remark: "",
});

const emptyThresholdRows = (): ThresholdFormRow[] =>
  thresholdDimensions.map((dimension) => ({
    ...dimension,
    threshold_value: "",
    threshold_unit: "GB",
    enabled: false,
  }));

function App() {
  const [activeTab, setActiveTab] = useState<TabKey>("overview");
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [profile, setProfile] = useState<AdminProfile | null>(null);

  const isSSHKeyMismatchError =
    error.includes("APP_MASTER_KEY") || error.includes("SSH 私钥无法解密") || error.includes("ssh key decrypt failed");

  const [loginForm, setLoginForm] = useState<LoginFormState>({
    username: "admin",
    password: "",
  });

  const [sshKeys, setSSHKeys] = useState<SSHKey[]>([]);
  const [machines, setMachines] = useState<Machine[]>([]);
  const [globalThresholds, setGlobalThresholds] = useState<ThresholdRule[]>([]);
  const [machineThresholds, setMachineThresholds] = useState<ThresholdRule[]>([]);
  const [notificationChannels, setNotificationChannels] = useState<NotificationChannel[]>([]);
  const [samples, setSamples] = useState<TrafficSample[]>([]);
  const [alerts, setAlerts] = useState<AlertItem[]>([]);

  const [selectedMachineID, setSelectedMachineID] = useState<number | null>(null);
  const [editingMachineID, setEditingMachineID] = useState<number | null>(null);
  const [machineForm, setMachineForm] = useState<MachineFormState>(emptyMachineForm());
  const [sshImportForm, setSSHImportForm] = useState<SSHKeyImportState>({ name: "", privateKey: "" });
  const [sshGenerateForm, setSSHGenerateForm] = useState<SSHKeyGenerateState>({ name: "" });
  const [globalThresholdForm, setGlobalThresholdForm] = useState<ThresholdFormRow[]>(emptyThresholdRows());
  const [machineThresholdForm, setMachineThresholdForm] = useState<ThresholdFormRow[]>(emptyThresholdRows());
  const [webhookForm, setWebhookForm] = useState<WebhookFormState>({
    enabled: false,
    method: "POST",
    url: "",
    headersText: "{}",
    bodyText: "",
  });
  const [telegramForm, setTelegramForm] = useState<TelegramFormState>({ enabled: false, botToken: "", chatID: "" });
  const [connectionResults, setConnectionResults] = useState<Record<number, ConnectionTestResponse>>({});
  const [collectResults, setCollectResults] = useState<CollectNowResponse["results"]>([]);
  const [webhookPreview, setWebhookPreview] = useState<WebhookPreviewState | null>(null);
  const [globalThresholdsSaved, setGlobalThresholdsSaved] = useState(true);
  const [machineThresholdsSaved, setMachineThresholdsSaved] = useState(true);
  const [webhookSaved, setWebhookSaved] = useState(true);
  const [telegramSaved, setTelegramSaved] = useState(true);
  const [machineFormSaved, setMachineFormSaved] = useState(true);

  const machineOptions = useMemo(
    () => machines.map((machine) => ({ value: machine.id, label: `${machine.name} (${machine.host})` })),
    [machines],
  );

  const selectedMachine = useMemo(
    () => machines.find((machine) => machine.id === selectedMachineID) ?? null,
    [machines, selectedMachineID],
  );

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    if (!profile) {
      return;
    }
    void loadProtectedData();
  }, [profile]);

  useEffect(() => {
    setMachineThresholdsSaved(false);
    if (!selectedMachineID || !profile) {
      setMachineThresholdForm(emptyThresholdRows());
      return;
    }
    void loadMachineThresholds(selectedMachineID);
  }, [selectedMachineID, profile]);

  async function bootstrap() {
    try {
      const nextProfile = await get<AdminProfile>("/api/v1/auth/profile");
      setProfile(nextProfile);
    } catch {
      setProfile(null);
      await loadPublicData();
    }
  }

  async function loadPublicData() {
    try {
      const [globalRules] = await Promise.all([get<ThresholdRule[]>("/api/v1/thresholds/global")]);
      setGlobalThresholds(globalRules);
      setGlobalThresholdForm(toThresholdFormRows(globalRules));
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    }
  }

  async function loadProtectedData() {
    setBusy(true);
    setError("");
    try {
      const [
        sshKeysResp,
        machinesResp,
        globalRules,
        channelsResp,
        samplesResp,
        alertsResp,
      ] = await Promise.all([
        get<SSHKey[]>("/api/v1/ssh-keys"),
        get<Machine[]>("/api/v1/machines"),
        get<ThresholdRule[]>("/api/v1/thresholds/global"),
        get<NotificationChannel[]>("/api/v1/notification-channels"),
        get<TrafficSampleList>("/api/v1/traffic-samples"),
        get<AlertList>("/api/v1/alerts"),
      ]);

      setSSHKeys(sshKeysResp);
      setMachines(machinesResp);
      setGlobalThresholds(globalRules);
      setGlobalThresholdForm(toThresholdFormRows(globalRules));
      setNotificationChannels(channelsResp);
      syncChannelForms(channelsResp);
      setSamples(samplesResp.items);
      setAlerts(alertsResp.items);

      const fallbackMachineID = selectedMachineID ?? machinesResp[0]?.id ?? null;
      setSelectedMachineID(fallbackMachineID);
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    } finally {
      setBusy(false);
    }
  }

  async function loadMachineThresholds(machineID: number) {
    try {
      const rules = await get<ThresholdRule[]>(`/api/v1/machines/${machineID}/thresholds`);
      setMachineThresholds(rules);
      setMachineThresholdForm(toThresholdFormRows(rules));
    } catch (loadError) {
      setError(toErrorMessage(loadError));
    }
  }

  function syncChannelForms(channels: NotificationChannel[]) {
    const webhook = channels.find((channel) => channel.channel_type === "webhook");
    const telegram = channels.find((channel) => channel.channel_type === "telegram");

    setWebhookForm((current) => ({
      ...current,
      enabled: webhook?.enabled ?? false,
      method: webhook?.method ?? "POST",
      url: webhook?.url ?? "",
      headersText: webhook?.headers ? JSON.stringify(webhook.headers, null, 2) : "{}",
      bodyText: webhook?.body ?? "",
    }));
    setWebhookSaved(true);
    setTelegramForm({
      enabled: telegram?.enabled ?? false,
      chatID: telegram?.chat_id ?? "",
      botToken: "",
    });
    setTelegramSaved(true);
  }

  async function handleLoginSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const nextProfile = await post<AdminProfile, { username: string; password: string }>("/api/v1/auth/login", {
        username: loginForm.username,
        password: loginForm.password,
      });
      setProfile(nextProfile);
      setLoginForm((current) => ({ ...current, password: "" }));
      setToast("登录成功");
    } catch (loginError) {
      setError(toErrorMessage(loginError));
    } finally {
      setBusy(false);
    }
  }

  async function handleLogout() {
    setBusy(true);
    try {
      await post<null>("/api/v1/auth/logout");
      setProfile(null);
      setToast("已退出登录");
    } catch (logoutError) {
      setError(toErrorMessage(logoutError));
    } finally {
      setBusy(false);
    }
  }

  async function handleImportSSHKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await post<SSHKey, { name: string; private_key: string }>("/api/v1/ssh-keys/import", {
        name: sshImportForm.name,
        private_key: sshImportForm.privateKey,
      });
      setSSHImportForm({ name: "", privateKey: "" });
      await loadProtectedData();
      setToast("SSH Key 导入成功");
    });
  }

  async function handleGenerateSSHKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await post<SSHKey, { name: string }>("/api/v1/ssh-keys/generate", {
        name: sshGenerateForm.name,
      });
      setSSHGenerateForm({ name: "" });
      await loadProtectedData();
      setToast("SSH Key 生成成功");
    });
  }

  async function handleDeleteSSHKey(id: number) {
    await submitAction(async () => {
      await del<null>(`/api/v1/ssh-keys/${id}`);
      await loadProtectedData();
      setToast("SSH Key 已删除");
    });
  }

  async function handleMachineSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const payload = {
      name: machineForm.name,
      host: machineForm.host,
      port: Number(machineForm.port),
      ssh_user: machineForm.sshUser,
      network_interface: machineForm.networkInterface,
      ssh_key_id: Number(machineForm.sshKeyID),
      collect_enabled: machineForm.collectEnabled,
      remark: machineForm.remark,
    };

    await submitAction(async () => {
      if (editingMachineID) {
        await patch<Machine, typeof payload>(`/api/v1/machines/${editingMachineID}`, payload);
        setToast("机器已更新");
      } else {
        await post<Machine, typeof payload>("/api/v1/machines", payload);
        setToast("机器已创建");
      }
      setEditingMachineID(null);
      setMachineForm(emptyMachineForm());
      setMachineFormSaved(true);
      await loadProtectedData();
    });
  }

  async function handleDeleteMachine(id: number) {
    await submitAction(async () => {
      await del<null>(`/api/v1/machines/${id}`);
      if (selectedMachineID === id) {
        setSelectedMachineID(null);
      }
      await loadProtectedData();
      setToast("机器已删除");
    });
  }

  async function handleTestConnection(id: number) {
    await submitAction(async () => {
      const result = await post<ConnectionTestResponse>(`/api/v1/machines/${id}/test-connection`);
      setConnectionResults((current) => ({ ...current, [id]: result }));
      setToast(`机器 ${id} 连通性验证完成`);
    });
  }

  async function handleSaveGlobalThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await put<null, { rules: ThresholdRulePayload[] }>("/api/v1/thresholds/global", {
        rules: toThresholdPayloads(globalThresholdForm),
      });
      await loadProtectedData();
      setGlobalThresholdsSaved(true);
      setToast("全局阈值已保存");
    });
  }

  async function handleSaveMachineThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedMachineID) {
      setError("请先选择机器");
      return;
    }

    await submitAction(async () => {
      await put<null, { rules: ThresholdRulePayload[] }>(`/api/v1/machines/${selectedMachineID}/thresholds`, {
        rules: toThresholdPayloads(machineThresholdForm),
      });
      await loadMachineThresholds(selectedMachineID);
      setMachineThresholdsSaved(true);
      setToast("单机阈值已保存");
    });
  }

  async function handleSaveWebhook(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await put<
        null,
        { enabled: boolean; method: "GET" | "POST"; url: string; headers: Record<string, string>; body: string }
      >("/api/v1/notification-channels/webhook", {
        enabled: webhookForm.enabled,
        method: webhookForm.method,
        url: webhookForm.url,
        headers: safeParseHeaders(webhookForm.headersText),
        body: webhookForm.bodyText,
      });
      setWebhookSaved(true);
      setToast("Webhook 渠道已保存");
    });
  }

  async function handleTestWebhook() {
    await submitAction(async () => {
      const parsedHeaders = safeParseHeaders(webhookForm.headersText);
      const response = await post<
        WebhookTestResponse,
        { method: "GET" | "POST"; url: string; headers: Record<string, string>; body: string }
      >("/api/v1/notification-channels/webhook/test", {
        method: webhookForm.method,
        url: webhookForm.url,
        headers: parsedHeaders,
        body: webhookForm.bodyText,
      });
      setWebhookPreview({
        url: response.rendered_url ?? renderWebhookPreviewTemplate(webhookForm.url),
        headersText: JSON.stringify(response.rendered_headers ?? renderWebhookPreviewHeaders(parsedHeaders), null, 2),
        bodyText: response.rendered_body ?? renderWebhookPreviewTemplate(webhookForm.bodyText),
      });
      setToast(response.body ? `Webhook 测试成功：${response.body}` : "Webhook 测试成功");
    });
  }

  async function handleSaveTelegram(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await put<null, { enabled: boolean; bot_token: string; chat_id: string }>(
        "/api/v1/notification-channels/telegram",
        {
          enabled: telegramForm.enabled,
          bot_token: telegramForm.botToken,
          chat_id: telegramForm.chatID,
        },
      );
      setTelegramForm((current) => ({ ...current, botToken: "" }));
      await loadProtectedData();
      setTelegramSaved(true);
      setToast("Telegram 渠道已保存");
    });
  }

  async function handleCollectNow(machineID?: number) {
    await submitAction(async () => {
      const response = await post<CollectNowResponse, { machine_id?: number }>("/api/v1/system/collect-now", {
        machine_id: machineID,
      });
      setCollectResults(response.results);
      await loadProtectedData();
      setToast("采集任务已执行");
    });
  }

  function startEditMachine(machine: Machine) {
    setEditingMachineID(machine.id);
    setMachineForm({
      name: machine.name,
      host: machine.host,
      port: String(machine.port),
      sshUser: machine.ssh_user,
      networkInterface: machine.network_interface,
      sshKeyID: String(machine.ssh_key_id),
      collectEnabled: machine.collect_enabled,
      remark: machine.remark,
    });
    setMachineFormSaved(true);
    setActiveTab("machines");
  }

  function resetMachineForm() {
    setEditingMachineID(null);
    setMachineForm(emptyMachineForm());
    setMachineFormSaved(true);
  }

  async function submitAction(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    setToast("");
    try {
      await action();
    } catch (submitError) {
      setError(toErrorMessage(submitError));
    } finally {
      setBusy(false);
    }
  }

  if (!profile) {
    return (
      <main className="app-shell auth-shell">
        <section className="panel auth-panel">
          <p className="eyebrow">traffic-monitor</p>
          <h1 className="panel-title">管理员登录</h1>
          <p className="muted">使用已初始化的管理员账号登录管理台。</p>
          <form className="form-grid" onSubmit={handleLoginSubmit}>
            <label className="field">
              <span>用户名</span>
              <input
                value={loginForm.username}
                onChange={(event) => setLoginForm((current) => ({ ...current, username: event.target.value }))}
                placeholder="admin"
              />
            </label>
            <label className="field">
              <span>密码</span>
              <input
                type="password"
                value={loginForm.password}
                onChange={(event) => setLoginForm((current) => ({ ...current, password: event.target.value }))}
                placeholder="请输入密码"
              />
            </label>
            <button className="primary-button" disabled={busy} type="submit">
              {busy ? "登录中..." : "登录"}
            </button>
          </form>
          {error ? <p className="message error">{error}</p> : null}
        </section>
      </main>
    );
  }

  return (
    <main className="console-shell">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">traffic-monitor</p>
          <h1 className="sidebar-title">流量监控后台</h1>
          <p className="muted">当前管理员：{profile.username}</p>
        </div>
        <nav className="tab-list">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              className={`tab-button${activeTab === tab.key ? " active" : ""}`}
              onClick={() => setActiveTab(tab.key)}
              type="button"
            >
              {tab.label}
            </button>
          ))}
        </nav>
        <button className="secondary-button" onClick={() => void handleLogout()} type="button">
          退出登录
        </button>
      </aside>

      <section className="content">
        <header className="content-header">
          <div>
            <h2>{tabTitle(activeTab)}</h2>
            <p className="muted">MVP 管理能力已接入当前后端接口。</p>
          </div>
          <div className="header-actions">
            <button className="secondary-button" onClick={() => void loadProtectedData()} type="button">
              刷新
            </button>
            <button className="primary-button" onClick={() => void handleCollectNow()} type="button">
              手动采集全部机器
            </button>
          </div>
        </header>

        {toast ? <div className="message success">{toast}</div> : null}
        {error ? (
          <div className={`message ${isSSHKeyMismatchError ? "warning" : "error"}`}>
            {isSSHKeyMismatchError ? (
              <>
                <strong>SSH Key 与当前 APP_MASTER_KEY 不匹配</strong>
                <span className="message-detail">{error}</span>
              </>
            ) : (
              error
            )}
          </div>
        ) : null}
        {busy ? <div className="message info">正在处理请求...</div> : null}

        {activeTab === "overview" ? (
          <OverviewTab
            sshKeys={sshKeys}
            machines={machines}
            notificationChannels={notificationChannels}
            samples={samples}
            alerts={alerts}
            collectResults={collectResults}
          />
        ) : null}

        {activeTab === "sshKeys" ? (
          <div className="grid two-columns">
            <section className="panel">
              <h3 className="panel-title">导入已有 SSH Key</h3>
              <form className="form-grid" onSubmit={handleImportSSHKey}>
                <label className="field">
                  <span>名称</span>
                  <input
                    value={sshImportForm.name}
                    onChange={(event) => setSSHImportForm((current) => ({ ...current, name: event.target.value }))}
                    placeholder="例如：prod-root"
                  />
                </label>
                <label className="field">
                  <span>私钥</span>
                  <textarea
                    rows={10}
                    value={sshImportForm.privateKey}
                    onChange={(event) =>
                      setSSHImportForm((current) => ({ ...current, privateKey: event.target.value }))
                    }
                    placeholder="粘贴 OpenSSH 私钥"
                  />
                </label>
                <button className="primary-button" disabled={busy} type="submit">
                  导入
                </button>
              </form>
            </section>

            <section className="panel">
              <h3 className="panel-title">生成新 Keypair</h3>
              <form className="form-grid" onSubmit={handleGenerateSSHKey}>
                <label className="field">
                  <span>名称</span>
                  <input
                    value={sshGenerateForm.name}
                    onChange={(event) => setSSHGenerateForm({ name: event.target.value })}
                    placeholder="例如：ops-generated"
                  />
                </label>
                <button className="primary-button" disabled={busy} type="submit">
                  生成
                </button>
              </form>

              <div className="list-block">
                <h4>SSH Key 列表</h4>
                {sshKeys.map((sshKey) => (
                  <article className="card" key={sshKey.id}>
                    <div className="card-header">
                      <strong>{sshKey.name}</strong>
                      <button className="danger-button" onClick={() => void handleDeleteSSHKey(sshKey.id)} type="button">
                        删除
                      </button>
                    </div>
                    <p className="card-meta">
                      类型：{sshKey.key_type} / 来源：{sshKey.source_type}
                    </p>
                    <p className="card-meta">指纹：{sshKey.fingerprint}</p>
                    <pre className="code-block">{sshKey.public_key}</pre>
                  </article>
                ))}
              </div>
            </section>
          </div>
        ) : null}

        {activeTab === "machines" ? (
          <div className="grid two-columns">
            <section className="panel">
              <div className="panel-header-inline">
                <h3 className="panel-title">{editingMachineID ? "编辑机器" : "新增机器"}</h3>
                {editingMachineID ? (
                  <button className="secondary-button" onClick={resetMachineForm} type="button">
                    取消编辑
                  </button>
                ) : null}
              </div>
              <form className="form-grid" onSubmit={handleMachineSubmit}>
                <label className="field">
                  <span>名称</span>
                  <input value={machineForm.name} onChange={(event) => updateMachineForm("name", event.target.value)} />
                </label>
                <label className="field">
                  <span>主机</span>
                  <input value={machineForm.host} onChange={(event) => updateMachineForm("host", event.target.value)} />
                </label>
                <label className="field">
                  <span>端口</span>
                  <input value={machineForm.port} onChange={(event) => updateMachineForm("port", event.target.value)} />
                </label>
                <label className="field">
                  <span>SSH 用户</span>
                  <input
                    value={machineForm.sshUser}
                    onChange={(event) => updateMachineForm("sshUser", event.target.value)}
                  />
                </label>
                <label className="field">
                  <span>网卡</span>
                  <input
                    value={machineForm.networkInterface}
                    onChange={(event) => updateMachineForm("networkInterface", event.target.value)}
                  />
                </label>
                <label className="field">
                  <span>SSH Key</span>
                  <select
                    value={machineForm.sshKeyID}
                    onChange={(event) => updateMachineForm("sshKeyID", event.target.value)}
                  >
                    <option value="">请选择 SSH Key</option>
                    {sshKeys.map((sshKey) => (
                      <option key={sshKey.id} value={sshKey.id}>
                        {sshKey.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field checkbox-field">
                  <input
                    checked={machineForm.collectEnabled}
                    onChange={(event) => updateMachineForm("collectEnabled", event.target.checked)}
                    type="checkbox"
                  />
                  <span>启用采集</span>
                </label>
                <label className="field full-width">
                  <span>备注</span>
                  <textarea
                    rows={3}
                    value={machineForm.remark}
                    onChange={(event) => updateMachineForm("remark", event.target.value)}
                  />
                </label>
                <button className="primary-button" disabled={busy || machineFormSaved} type="submit">
                  {editingMachineID ? "保存修改" : "创建机器"}
                </button>
              </form>
            </section>

            <section className="panel">
              <h3 className="panel-title">机器列表</h3>
              <div className="table-wrapper">
                <table>
                  <thead>
                    <tr>
                      <th>名称</th>
                      <th>主机</th>
                      <th>网卡</th>
                      <th>SSH Key</th>
                      <th>采集</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {machines.map((machine) => (
                      <tr key={machine.id}>
                        <td>{machine.name}</td>
                        <td>{machine.host}:{machine.port}</td>
                        <td>{machine.network_interface}</td>
                        <td>{machine.ssh_key_id}</td>
                        <td>{machine.collect_enabled ? "启用" : "停用"}</td>
                        <td>
                          <div className="action-row">
                            <button className="secondary-button" onClick={() => startEditMachine(machine)} type="button">
                              编辑
                            </button>
                            <button className="secondary-button" onClick={() => void handleTestConnection(machine.id)} type="button">
                              测试
                            </button>
                            <button className="danger-button" onClick={() => void handleDeleteMachine(machine.id)} type="button">
                              删除
                            </button>
                          </div>
                          {connectionResults[machine.id] ? (
                            <p className="card-meta">
                              测试结果：{connectionResults[machine.id].status}
                              {connectionResults[machine.id].vnstat_version
                                ? ` / ${connectionResults[machine.id].vnstat_version}`
                                : ""}
                            </p>
                          ) : null}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>
        ) : null}

        {activeTab === "thresholds" ? (
          <div className="grid two-columns">
            <section className="panel">
              <h3 className="panel-title">全局阈值</h3>
              <form onSubmit={handleSaveGlobalThresholds}>
                <ThresholdEditor
                  rows={globalThresholdForm}
                  onChange={(rows) => {
                    setGlobalThresholdForm(rows);
                    setGlobalThresholdsSaved(false);
                  }}
                />
                <button className="primary-button" disabled={busy || globalThresholdsSaved} type="submit">
                  保存全局阈值
                </button>
              </form>
            </section>

            <section className="panel">
              <div className="panel-header-inline">
                <h3 className="panel-title">单机覆盖阈值</h3>
                <select
                  value={selectedMachineID ?? ""}
                  onChange={(event) => setSelectedMachineID(event.target.value ? Number(event.target.value) : null)}
                >
                  <option value="">请选择机器</option>
                  {machineOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
              {selectedMachine ? (
                <form onSubmit={handleSaveMachineThresholds}>
                  <ThresholdEditor
                    rows={machineThresholdForm}
                    onChange={(rows) => {
                      setMachineThresholdForm(rows);
                      setMachineThresholdsSaved(false);
                    }}
                  />
                  <button className="primary-button" disabled={busy || machineThresholdsSaved} type="submit">
                    保存 {selectedMachine.name} 的阈值
                  </button>
                </form>
              ) : (
                <p className="muted">请先选择机器。</p>
              )}
            </section>
          </div>
        ) : null}

        {activeTab === "notifications" ? (
          <div className="grid two-columns">
            <section className="panel">
              <h3 className="panel-title">Webhook 通知</h3>
              <form className="form-grid" onSubmit={handleSaveWebhook}>
                <label className="field checkbox-field">
                  <input
                    checked={webhookForm.enabled}
                    onChange={(event) => {
                      setWebhookForm((current) => ({ ...current, enabled: event.target.checked }));
                      setWebhookSaved(false);
                    }}
                    type="checkbox"
                  />
                  <span>启用 Webhook</span>
                </label>
                <label className="field">
                  <span>请求方式</span>
                  <select
                    value={webhookForm.method}
                    onChange={(event) => {
                      setWebhookForm((current) => ({ ...current, method: event.target.value as "GET" | "POST" }));
                      setWebhookSaved(false);
                    }}
                  >
                    <option value="POST">POST</option>
                    <option value="GET">GET</option>
                  </select>
                </label>
                <label className="field">
                  <span>URL</span>
                  <input
                    value={webhookForm.url}
                    onChange={(event) => {
                      setWebhookForm((current) => ({ ...current, url: event.target.value }));
                      setWebhookSaved(false);
                    }}
                    placeholder="https://example.com/hook"
                  />
                </label>
                <label className="field full-width">
                  <span>Headers(JSON 模板)</span>
                  <textarea
                    rows={5}
                    value={webhookForm.headersText}
                    onChange={(event) => {
                      setWebhookForm((current) => ({ ...current, headersText: event.target.value }));
                      setWebhookSaved(false);
                    }}
                    placeholder={`{
  "Authorization": "Bearer {{alert_key}}",
  "X-Metric": "{{metric_type}}",
  "X-Machine-Name": "{{machine_name}}"
}`}
                  />
                </label>
                <label className="field full-width">
                  <span>Body 模板</span>
                  <textarea
                    rows={8}
                    value={webhookForm.bodyText}
                    onChange={(event) => {
                      setWebhookForm((current) => ({ ...current, bodyText: event.target.value }));
                      setWebhookSaved(false);
                    }}
                    placeholder={`{
  "machine_id": "{{machine_id}}",
  "machine_name": "{{machine_name}}",
  "machine_host": "{{machine_host}}",
  "period_type": "{{period_type}}",
  "metric_type": "{{metric_type}}",
  "bucket_time": "{{bucket_time}}",
  "threshold_mb": "{{threshold_mb}}",
  "actual_mb": "{{actual_mb}}",
  "alert_key": "{{alert_key}}"
}`}
                  />
                </label>
                <div className="card">
                  <div className="card-header">
                    <strong>可用变量</strong>
                  </div>
                  <p className="card-meta">
                    URL、Headers、Body 都支持以下变量模板：
                    <code> {"{{machine_id}}"}</code>
                    <code> {"{{machine_name}}"}</code>
                    <code> {"{{machine_host}}"}</code>
                    <code> {"{{period_type}}"}</code>
                    <code> {"{{metric_type}}"}</code>
                    <code> {"{{bucket_time}}"}</code>
                    <code> {"{{threshold_mb}}"}</code>
                    <code> {"{{actual_mb}}"}</code>
                    <code> {"{{alert_key}}"}</code>
                  </p>
                </div>
                {webhookPreview ? (
                  <div className="card">
                    <div className="card-header">
                      <strong>渲染预览</strong>
                    </div>
                    <p className="card-meta">URL</p>
                    <pre className="code-block">{webhookPreview.url || "-"}</pre>
                    <p className="card-meta">Headers</p>
                    <pre className="code-block">{webhookPreview.headersText || "{}"}</pre>
                    <p className="card-meta">Body</p>
                    <pre className="code-block">{webhookPreview.bodyText || "-"}</pre>
                  </div>
                ) : null}
                <div className="action-row">
                  <button className="secondary-button" disabled={busy} onClick={() => void handleTestWebhook()} type="button">
                    测试 Webhook
                  </button>
                  <button className="primary-button" disabled={busy || webhookSaved} type="submit">
                    保存 Webhook
                  </button>
                </div>
              </form>
            </section>

            <section className="panel">
              <h3 className="panel-title">Telegram 通知</h3>
              <form className="form-grid" onSubmit={handleSaveTelegram}>
                <label className="field checkbox-field">
                  <input
                    checked={telegramForm.enabled}
                    onChange={(event) => {
                      setTelegramForm((current) => ({ ...current, enabled: event.target.checked }));
                      setTelegramSaved(false);
                    }}
                    type="checkbox"
                  />
                  <span>启用 Telegram</span>
                </label>
                <label className="field">
                  <span>Bot Token</span>
                  <input
                    value={telegramForm.botToken}
                    onChange={(event) => {
                      setTelegramForm((current) => ({ ...current, botToken: event.target.value }));
                      setTelegramSaved(false);
                    }}
                    placeholder="仅保存时填写"
                  />
                </label>
                <label className="field">
                  <span>Chat ID</span>
                  <input
                    value={telegramForm.chatID}
                    onChange={(event) => {
                      setTelegramForm((current) => ({ ...current, chatID: event.target.value }));
                      setTelegramSaved(false);
                    }}
                  />
                </label>
                <button className="primary-button" disabled={busy || telegramSaved} type="submit">
                  保存 Telegram
                </button>
              </form>

              <div className="list-block">
                <h4>当前渠道状态</h4>
                {notificationChannels.map((channel) => (
                  <article className="card" key={channel.channel_type}>
                    <div className="card-header">
                      <strong>{channel.channel_type}</strong>
                      <span className={`status-badge ${channel.enabled ? "ok" : "idle"}`}>
                        {channel.enabled ? "启用" : "停用"}
                      </span>
                    </div>
                    <p className="card-meta">已配置：{channel.configured ? "是" : "否"}</p>
                    {channel.url ? <p className="card-meta">URL：{channel.url}</p> : null}
                    {channel.chat_id ? <p className="card-meta">Chat ID：{channel.chat_id}</p> : null}
                    {channel.token_masked ? <p className="card-meta">Token：{channel.token_masked}</p> : null}
                  </article>
                ))}
              </div>
            </section>
          </div>
        ) : null}

        {activeTab === "samples" ? (
          <section className="panel">
            <div className="panel-header-inline">
              <h3 className="panel-title">流量样本</h3>
              <div className="header-actions">
                {selectedMachineID ? (
                  <button className="secondary-button" onClick={() => void handleCollectNow(selectedMachineID)} type="button">
                    采集当前机器
                  </button>
                ) : null}
                <select
                  value={selectedMachineID ?? ""}
                  onChange={(event) => setSelectedMachineID(event.target.value ? Number(event.target.value) : null)}
                >
                  <option value="">全部机器</option>
                  {machineOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="table-wrapper">
              <table>
                <thead>
                  <tr>
                    <th>机器</th>
                    <th>周期</th>
                    <th>桶时间</th>
                    <th>上行</th>
                    <th>下行</th>
                    <th>总量</th>
                    <th>采集时间</th>
                  </tr>
                </thead>
                <tbody>
                  {samples
                    .filter((sample) => !selectedMachineID || sample.machine_id === selectedMachineID)
                    .map((sample) => (
                      <tr key={`${sample.id}-${sample.period_type}`}>
                        <td>{sample.machine_id}</td>
                        <td>{sample.period_type}</td>
                        <td>{formatTime(sample.bucket_time)}</td>
                        <td>{formatTrafficValue(sample.upload_mb)}</td>
                        <td>{formatTrafficValue(sample.download_mb)}</td>
                        <td>{formatTrafficValue(sample.total_mb)}</td>
                        <td>{formatTime(sample.collected_at)}</td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>

            {collectResults.length > 0 ? (
              <div className="list-block">
                <h4>最近手动采集结果</h4>
                {collectResults.map((result) => (
                  <article className="card" key={`${result.machine_id}-${result.status}`}>
                    <div className="card-header">
                      <strong>机器 {result.machine_id}</strong>
                      <span className={`status-badge ${result.status === "success" ? "ok" : "error"}`}>
                        {result.status}
                      </span>
                    </div>
                    <p className="card-meta">样本数：{result.sample_count}</p>
                    {result.error ? <p className="card-meta">错误：{result.error}</p> : null}
                  </article>
                ))}
              </div>
            ) : null}
          </section>
        ) : null}

        {activeTab === "alerts" ? (
          <section className="panel">
            <h3 className="panel-title">告警记录</h3>
            <div className="table-wrapper">
              <table>
                <thead>
                  <tr>
                    <th>机器</th>
                    <th>周期</th>
                    <th>维度</th>
                    <th>告警周期</th>
                    <th>阈值</th>
                    <th>实际</th>
                    <th>通知状态</th>
                    <th>通知时间</th>
                  </tr>
                </thead>
                <tbody>
                  {alerts.map((alert) => (
                    <tr key={alert.id}>
                      <td>{alert.machine_id}</td>
                      <td>{alert.period_type}</td>
                      <td>{alert.metric_type}</td>
                      <td>{formatAlertPeriod(alert.period_type, alert.bucket_time)}</td>
                      <td>{formatTrafficValue(alert.threshold_mb)}</td>
                      <td>{formatTrafficValue(alert.actual_mb)}</td>
                      <td>{alert.notify_status}</td>
                      <td>{alert.notified_at ? formatTime(alert.notified_at) : "-"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ) : null}
      </section>
    </main>
  );

  function updateMachineForm<Key extends keyof MachineFormState>(key: Key, value: MachineFormState[Key]) {
    setMachineForm((current) => ({ ...current, [key]: value }));
    setMachineFormSaved(false);
  }
}

function OverviewTab(props: {
  sshKeys: SSHKey[];
  machines: Machine[];
  notificationChannels: NotificationChannel[];
  samples: TrafficSample[];
  alerts: AlertItem[];
  collectResults: CollectNowResponse["results"];
}) {
  const enabledMachines = props.machines.filter((machine) => machine.collect_enabled).length;
  const enabledChannels = props.notificationChannels.filter((channel) => channel.enabled).length;

  return (
    <div className="grid overview-grid">
      <StatCard label="SSH Key" value={String(props.sshKeys.length)} help="当前可用的登录密钥数量" />
      <StatCard label="机器总数" value={String(props.machines.length)} help={`启用采集 ${enabledMachines} 台`} />
      <StatCard label="通知渠道" value={String(enabledChannels)} help="已启用的通知渠道数量" />
      <StatCard label="最近样本" value={String(props.samples.length)} help="当前查询到的样本条数" />
      <StatCard label="告警总数" value={String(props.alerts.length)} help="当前查询到的告警条数" />
      <StatCard
        label="最近采集执行"
        value={props.collectResults.length ? props.collectResults[0].status : "未执行"}
        help="手动采集的最近一次结果"
      />
    </div>
  );
}

function StatCard(props: { label: string; value: string; help: string }) {
  return (
    <section className="panel stat-card">
      <p className="muted">{props.label}</p>
      <h3>{props.value}</h3>
      <p className="card-meta">{props.help}</p>
    </section>
  );
}

function ThresholdEditor(props: {
  rows: ThresholdFormRow[];
  onChange: (rows: ThresholdFormRow[]) => void;
}) {
  return (
    <div className="table-wrapper threshold-editor">
      <table>
        <thead>
          <tr>
            <th>周期</th>
            <th>维度</th>
            <th>阈值</th>
            <th>单位</th>
            <th>启用</th>
            <th>来源</th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row, index) => (
            <tr key={`${row.period_type}-${row.metric_type}`}>
              <td>{row.period_type}</td>
              <td>{row.metric_type}</td>
              <td>
                <input
                  value={row.threshold_value}
                  onChange={(event) => updateRow(props.rows, index, "threshold_value", event.target.value, props.onChange)}
                />
              </td>
              <td>
                <select
                  value={row.threshold_unit}
                  onChange={(event) =>
                    updateRow(
                      props.rows,
                      index,
                      "threshold_unit",
                      event.target.value as "MB" | "GB",
                      props.onChange,
                    )
                  }
                >
                  <option value="MB">MB</option>
                  <option value="GB">GB</option>
                </select>
              </td>
              <td>
                <input
                  checked={row.enabled}
                  onChange={(event) => updateRow(props.rows, index, "enabled", event.target.checked, props.onChange)}
                  type="checkbox"
                />
              </td>
              <td>{row.source ?? "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function updateRow<Key extends keyof ThresholdFormRow>(
  rows: ThresholdFormRow[],
  index: number,
  key: Key,
  value: ThresholdFormRow[Key],
  onChange: (rows: ThresholdFormRow[]) => void,
) {
  const nextRows = rows.map((row, rowIndex) => (rowIndex === index ? { ...row, [key]: value } : row));
  onChange(nextRows);
}

type ThresholdRulePayload = {
  period_type: string;
  metric_type: string;
  threshold_value: number;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
};

function toThresholdPayloads(rows: ThresholdFormRow[]): ThresholdRulePayload[] {
  return rows.map((row) => ({
    period_type: row.period_type,
    metric_type: row.metric_type,
    threshold_value: Number(row.threshold_value || "0"),
    threshold_unit: row.threshold_unit,
    enabled: row.enabled,
  }));
}

function toThresholdFormRows(rules: ThresholdRule[]): ThresholdFormRow[] {
  return thresholdDimensions.map((dimension) => {
    const matched = rules.find(
      (rule) => rule.period_type === dimension.period_type && rule.metric_type === dimension.metric_type,
    );

    return {
      ...dimension,
      threshold_value: matched ? String(matched.threshold_value) : "",
      threshold_unit: (matched?.threshold_unit as "MB" | "GB") || "GB",
      enabled: matched?.enabled ?? false,
      source: matched?.source,
    };
  });
}

function safeParseHeaders(value: string): Record<string, string> {
  try {
    const parsed = JSON.parse(value) as Record<string, string>;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function renderWebhookPreviewTemplate(value: string): string {
  return [
    ["{{machine_id}}", "1"],
    ["{{machine_name}}", "test-machine"],
    ["{{machine_host}}", "127.0.0.1"],
    ["{{period_type}}", "hourly"],
    ["{{metric_type}}", "total"],
    ["{{bucket_time}}", "2026-04-25 12:00:00 +0000 UTC"],
    ["{{bucket_time_rfc3339}}", "2026-04-25T12:00:00Z"],
    ["{{threshold_mb}}", "1024"],
    ["{{actual_mb}}", "1536"],
    ["{{alert_key}}", "test:webhook:alert"],
  ].reduce((result, [token, replacement]) => result.split(token).join(replacement), value);
}

function renderWebhookPreviewHeaders(headers: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [key, renderWebhookPreviewTemplate(value)]),
  );
}

function toErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}

function formatTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}

function formatNumber(value: number) {
  return value.toFixed(3);
}

function formatTrafficValue(valueMB: number) {
  if (valueMB >= 1024 * 1024) {
    return `${(valueMB / (1024 * 1024)).toFixed(3)} TB`;
  }

  if (valueMB >= 1024) {
    return `${(valueMB / 1024).toFixed(3)} GB`;
  }

  return `${valueMB.toFixed(3)} MB`;
}

function formatAlertPeriod(periodType: string, bucketTime: string) {
  const start = new Date(bucketTime);

  if (Number.isNaN(start.getTime())) {
    return bucketTime;
  }

  if (periodType === "hourly") {
    const end = new Date(start.getTime() + 60 * 60 * 1000 - 1000);
    return `${formatTime(bucketTime)} - ${end.toLocaleString("zh-CN", { hour12: false })}`;
  }

  if (periodType === "daily") {
    return `${start.toLocaleDateString("zh-CN")} 全天`;
  }

  return formatTime(bucketTime);
}

function tabTitle(activeTab: TabKey) {
  return tabs.find((tab) => tab.key === activeTab)?.label ?? "管理台";
}

export default App;
