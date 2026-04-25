import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { del, get, patch, post, put } from "./api";
import OverviewTab from "./components/OverviewTab";
import { useI18n } from "./lib/i18n";
import type {
  LoginFormState,
  MachineFormState,
  SSHKeyGenerateState,
  SSHKeyImportState,
  TabKey,
  TelegramFormState,
  ThresholdFormRow,
  WebhookFormState,
  WebhookPreviewState,
} from "./lib/app-types";
import {
  emptyThresholdRows,
  safeParseHeaders,
  tabKeyFromPath,
  tabPath,
  tabs,
  tabTitle,
  toErrorMessage,
  toThresholdFormRows,
  toThresholdPayloads,
  renderWebhookPreviewHeaders,
  renderWebhookPreviewTemplate,
} from "./lib/app-utils";
import SSHKeysPage from "./pages/SSHKeysPage";
import MachinesPage from "./pages/MachinesPage";
import ThresholdsPage from "./pages/ThresholdsPage";
import NotificationsPage from "./pages/NotificationsPage";
import SamplesPage from "./pages/SamplesPage";
import AlertsPage from "./pages/AlertsPage";
import type {
  AdminProfile,
  AlertItem,
  AlertList,
  CleanupHistoryResponse,
  CollectNowResponse,
  ConnectionTestResponse,
  Machine,
  NotificationChannel,
  SSHKey,
  ThresholdRule,
  TrafficSample,
  TrafficSampleList,
  WebhookTestResponse,
} from "./types";

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

function App() {
  const { language, setLanguage, t } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const activeTab = tabKeyFromPath(location.pathname);

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
  const [selectedAlertMachineID, setSelectedAlertMachineID] = useState<number | null>(null);
  const [editingMachineID, setEditingMachineID] = useState<number | null>(null);
  const [machineForm, setMachineForm] = useState<MachineFormState>(emptyMachineForm());
  const [sshImportForm, setSSHImportForm] = useState<SSHKeyImportState>({ name: "", privateKey: "" });
  const [sshGenerateForm, setSSHGenerateForm] = useState<SSHKeyGenerateState>({ name: "" });
  const [renamingSSHKeyID, setRenamingSSHKeyID] = useState<number | null>(null);
  const [sshRenameName, setSSHRenameName] = useState("");
  const [globalThresholdForm, setGlobalThresholdForm] = useState<ThresholdFormRow[]>(emptyThresholdRows());
  const [machineThresholdForm, setMachineThresholdForm] = useState<ThresholdFormRow[]>(emptyThresholdRows());
  const [webhookForm, setWebhookForm] = useState<WebhookFormState>({
    enabled: false,
    method: "POST",
    url: "",
    headersText: "{}",
    bodyText: "",
  });
  const [telegramForm, setTelegramForm] = useState<TelegramFormState>({
    enabled: false,
    botToken: "",
    chatID: "",
  });
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
    setMachineThresholdsSaved(true);
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
      setError(toErrorMessage(loadError, language));
    }
  }

  async function loadProtectedData() {
    setBusy(true);
    setError("");
    try {
      const [sshKeysResp, machinesResp, globalRules, channelsResp, samplesResp, alertsResp] = await Promise.all([
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

      setSelectedMachineID((currentSelectedMachineID) => {
        if (currentSelectedMachineID === null) {
          return null;
        }

        return machinesResp.some((machine) => machine.id === currentSelectedMachineID)
          ? currentSelectedMachineID
          : machinesResp[0]?.id ?? null;
      });
    } catch (loadError) {
      setError(toErrorMessage(loadError, language));
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
      setError(toErrorMessage(loadError, language));
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
      setToast(t("loginSuccess"));
    } catch (loginError) {
      setError(toErrorMessage(loginError, language));
    } finally {
      setBusy(false);
    }
  }

  async function handleLogout() {
    setBusy(true);
    try {
      await post<null>("/api/v1/auth/logout");
      setProfile(null);
      navigate("/overview", { replace: true });
      setToast(t("logoutSuccess"));
    } catch (logoutError) {
      setError(toErrorMessage(logoutError, language));
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
      setToast(t("sshKeyImportSuccess"));
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
      setToast(t("sshKeyGenerateSuccess"));
    });
  }

  async function handleDeleteSSHKey(id: number) {
    await submitAction(async () => {
      await del<null>(`/api/v1/ssh-keys/${id}`);
      if (renamingSSHKeyID === id) {
        setRenamingSSHKeyID(null);
        setSSHRenameName("");
      }
      await loadProtectedData();
      setToast(t("sshKeyDeleteSuccess"));
    });
  }

  function startRenameSSHKey(sshKey: SSHKey) {
    setRenamingSSHKeyID(sshKey.id);
    setSSHRenameName(sshKey.name);
  }

  function cancelRenameSSHKey() {
    setRenamingSSHKeyID(null);
    setSSHRenameName("");
  }

  async function handleRenameSSHKey(id: number) {
    await submitAction(async () => {
      await patch<SSHKey, { name: string }>(`/api/v1/ssh-keys/${id}`, {
        name: sshRenameName,
      });
      setRenamingSSHKeyID(null);
      setSSHRenameName("");
      await loadProtectedData();
      setToast(t("sshKeyRenameSuccess"));
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
        setToast(t("machineUpdateSuccess"));
      } else {
        await post<Machine, typeof payload>("/api/v1/machines", payload);
        setToast(t("machineCreateSuccess"));
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
      setToast(t("machineDeleteSuccess"));
    });
  }

  async function handleTestConnection(id: number) {
    await submitAction(async () => {
      const result = await post<ConnectionTestResponse>(`/api/v1/machines/${id}/test-connection`);
      setConnectionResults((current) => ({ ...current, [id]: result }));
      setToast(t("machineConnectionDone", { id }));
    });
  }

  async function handleSaveGlobalThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await put<null, { rules: ReturnType<typeof toThresholdPayloads> }>("/api/v1/thresholds/global", {
        rules: toThresholdPayloads(globalThresholdForm),
      });
      await loadProtectedData();
      setGlobalThresholdsSaved(true);
      setToast(t("globalThresholdSaveSuccess"));
    });
  }

  async function handleSaveMachineThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedMachineID) {
      setError(t("selectMachineFirst"));
      return;
    }

    await submitAction(async () => {
      await put<null, { rules: ReturnType<typeof toThresholdPayloads> }>(
        `/api/v1/machines/${selectedMachineID}/thresholds`,
        {
          rules: toThresholdPayloads(machineThresholdForm),
        },
      );
      await loadMachineThresholds(selectedMachineID);
      setMachineThresholdsSaved(true);
      setToast(t("machineThresholdSaveSuccess"));
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
      setToast(t("webhookSaveSuccess"));
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
      setToast(response.body ? t("webhookTestSuccessWithBody", { body: response.body }) : t("webhookTestSuccess"));
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
      setToast(t("telegramSaveSuccess"));
    });
  }

  async function handleCollectNow(machineID?: number) {
    await submitAction(async () => {
      const response = await post<CollectNowResponse, { machine_id?: number }>("/api/v1/system/collect-now", {
        machine_id: machineID,
      });
      setCollectResults(response.results);
      await loadProtectedData();
      setToast(t("collectNowSuccess"));
    });
  }

  async function handleCleanupHistory() {
    if (!window.confirm(t("cleanupHistoryConfirm"))) {
      return;
    }

    await submitAction(async () => {
      const response = await post<
        CleanupHistoryResponse,
        { delete_samples: boolean; delete_alerts: boolean }
      >("/api/v1/traffic-samples/cleanup", {
        delete_samples: true,
        delete_alerts: true,
      });
      await loadProtectedData();
      setToast(t("cleanupHistoryDone", { samples: response.deleted_samples, alerts: response.deleted_alerts }));
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
    navigate("/machines");
  }

  function resetMachineForm() {
    setEditingMachineID(null);
    setMachineForm(emptyMachineForm());
    setMachineFormSaved(true);
  }

  function updateMachineForm<Key extends keyof MachineFormState>(key: Key, value: MachineFormState[Key]) {
    setMachineForm((current) => ({ ...current, [key]: value }));
    setMachineFormSaved(false);
  }

  function updateWebhookForm(updater: (current: WebhookFormState) => WebhookFormState) {
    setWebhookForm((current) => updater(current));
    setWebhookSaved(false);
  }

  function updateTelegramForm(updater: (current: TelegramFormState) => TelegramFormState) {
    setTelegramForm((current) => updater(current));
    setTelegramSaved(false);
  }

  async function submitAction(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    setToast("");
    try {
      await action();
    } catch (submitError) {
      setError(toErrorMessage(submitError, language));
    } finally {
      setBusy(false);
    }
  }

  if (!profile) {
    return (
      <main className="app-shell auth-shell">
        <section className="panel auth-panel">
          <p className="eyebrow">traffic-monitor</p>
          <div className="panel-header-inline auth-header">
            <div>
              <h1 className="panel-title">{t("loginTitle")}</h1>
              <p className="muted">{t("loginDescription")}</p>
            </div>
            <div className="language-switcher" aria-label={t("languageSwitcherLabel")}>
              <button
                className={`secondary-button language-button${language === "zh" ? " active" : ""}`}
                onClick={() => setLanguage("zh")}
                type="button"
              >
                {t("languageChinese")}
              </button>
              <button
                className={`secondary-button language-button${language === "en" ? " active" : ""}`}
                onClick={() => setLanguage("en")}
                type="button"
              >
                {t("languageEnglish")}
              </button>
            </div>
          </div>
          <form className="form-grid" onSubmit={handleLoginSubmit}>
            <label className="field">
              <span>{t("username")}</span>
              <input
                value={loginForm.username}
                onChange={(event) => setLoginForm((current) => ({ ...current, username: event.target.value }))}
                placeholder="admin"
              />
            </label>
            <label className="field">
              <span>{t("password")}</span>
              <input
                type="password"
                value={loginForm.password}
                onChange={(event) => setLoginForm((current) => ({ ...current, password: event.target.value }))}
                placeholder={t("passwordPlaceholder")}
              />
            </label>
            <button className="primary-button" disabled={busy} type="submit">
              {busy ? t("loggingIn") : t("login")}
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
          <h1 className="sidebar-title">{t("sidebarTitle")}</h1>
          <p className="muted">{t("currentAdmin", { username: profile.username })}</p>
        </div>

        <div className="language-switcher" aria-label={t("languageSwitcherLabel")}>
          <button
            className={`secondary-button language-button${language === "zh" ? " active" : ""}`}
            onClick={() => setLanguage("zh")}
            type="button"
          >
            {t("languageChinese")}
          </button>
          <button
            className={`secondary-button language-button${language === "en" ? " active" : ""}`}
            onClick={() => setLanguage("en")}
            type="button"
          >
            {t("languageEnglish")}
          </button>
        </div>

        <nav className="tab-list">
          {tabs.map((tab) => (
            <NavLink
              key={tab.key}
              to={tab.path}
              className={({ isActive }) => `tab-link${isActive ? " active" : ""}`}
            >
              {tabTitle(tab.key, language)}
            </NavLink>
          ))}
        </nav>

        <button className="secondary-button" onClick={() => void handleLogout()} type="button">
          {t("logout")}
        </button>
      </aside>

      <section className="content">
        <header className="content-header">
          <div>
            <h2>{tabTitle(activeTab, language)}</h2>
            <p className="muted">{t("dashboardDescription")}</p>
          </div>
          <div className="header-actions">
            <button className="secondary-button" onClick={() => void loadProtectedData()} type="button">
              {t("refresh")}
            </button>
            <button className="secondary-button" onClick={() => void handleCleanupHistory()} type="button">
              {t("cleanupHistory")}
            </button>
            <button className="primary-button" onClick={() => void handleCollectNow()} type="button">
              {t("collectAllMachines")}
            </button>
          </div>
        </header>

        {toast ? <div className="message success">{toast}</div> : null}
        {error ? (
          <div className={`message ${isSSHKeyMismatchError ? "warning" : "error"}`}>
            {isSSHKeyMismatchError ? (
              <>
                <strong>{t("sshKeyMismatchTitle")}</strong>
                <span className="message-detail">{error}</span>
              </>
            ) : (
              error
            )}
          </div>
        ) : null}
        {busy ? <div className="message info">{t("processingRequest")}</div> : null}

        <Routes>
          <Route path="/" element={<Navigate replace to="/overview" />} />
          <Route
            path="/overview"
            element={
              <OverviewTab
                sshKeys={sshKeys}
                machines={machines}
                notificationChannels={notificationChannels}
                samples={samples}
                alerts={alerts}
                collectResults={collectResults}
                onNavigate={(tab) => navigate(tabPath(tab as TabKey))}
              />
            }
          />
          <Route
            path="/ssh-keys"
            element={
              <SSHKeysPage
                busy={busy}
                sshKeys={sshKeys}
                sshImportForm={sshImportForm}
                sshGenerateForm={sshGenerateForm}
                setSSHImportForm={setSSHImportForm}
                setSSHGenerateForm={setSSHGenerateForm}
                onImportSubmit={handleImportSSHKey}
                onGenerateSubmit={handleGenerateSSHKey}
                onDeleteSSHKey={handleDeleteSSHKey}
                renamingSSHKeyID={renamingSSHKeyID}
                sshRenameName={sshRenameName}
                setSSHRenameName={setSSHRenameName}
                onStartRenameSSHKey={startRenameSSHKey}
                onCancelRenameSSHKey={cancelRenameSSHKey}
                onRenameSSHKey={handleRenameSSHKey}
              />
            }
          />
          <Route
            path="/machines"
            element={
              <MachinesPage
                busy={busy}
                editingMachineID={editingMachineID}
                machineForm={machineForm}
                machineFormSaved={machineFormSaved}
                sshKeys={sshKeys}
                machines={machines}
                connectionResults={connectionResults}
                onMachineSubmit={handleMachineSubmit}
                onResetMachineForm={resetMachineForm}
                onUpdateMachineForm={updateMachineForm}
                onStartEditMachine={startEditMachine}
                onTestConnection={handleTestConnection}
                onDeleteMachine={handleDeleteMachine}
              />
            }
          />
          <Route
            path="/thresholds"
            element={
              <ThresholdsPage
                busy={busy}
                globalThresholdForm={globalThresholdForm}
                machineThresholdForm={machineThresholdForm}
                globalThresholdsSaved={globalThresholdsSaved}
                machineThresholdsSaved={machineThresholdsSaved}
                selectedMachineID={selectedMachineID}
                selectedMachine={selectedMachine}
                machineOptions={machineOptions}
                onSelectMachine={setSelectedMachineID}
                onChangeGlobalThresholdForm={(rows) => {
                  setGlobalThresholdForm(rows);
                  setGlobalThresholdsSaved(false);
                }}
                onChangeMachineThresholdForm={(rows) => {
                  setMachineThresholdForm(rows);
                  setMachineThresholdsSaved(false);
                }}
                onSaveGlobalThresholds={handleSaveGlobalThresholds}
                onSaveMachineThresholds={handleSaveMachineThresholds}
              />
            }
          />
          <Route
            path="/notifications"
            element={
              <NotificationsPage
                busy={busy}
                notificationChannels={notificationChannels}
                webhookForm={webhookForm}
                webhookSaved={webhookSaved}
                webhookPreview={webhookPreview}
                telegramForm={telegramForm}
                telegramSaved={telegramSaved}
                onWebhookFormChange={updateWebhookForm}
                onTelegramFormChange={updateTelegramForm}
                onSaveWebhook={handleSaveWebhook}
                onTestWebhook={() => void handleTestWebhook()}
                onSaveTelegram={handleSaveTelegram}
              />
            }
          />
          <Route
            path="/samples"
            element={
              <SamplesPage
                selectedMachineID={selectedMachineID}
                machineOptions={machineOptions}
                samples={samples}
                collectResults={collectResults}
                onSelectMachine={setSelectedMachineID}
                onCollectCurrentMachine={(machineID) => void handleCollectNow(machineID)}
              />
            }
          />
          <Route
            path="/alerts"
            element={
              <AlertsPage
                alerts={alerts}
                machineOptions={machineOptions}
                selectedMachineID={selectedAlertMachineID}
                onSelectMachine={setSelectedAlertMachineID}
              />
            }
          />
          <Route path="*" element={<Navigate replace to="/overview" />} />
        </Routes>
      </section>
    </main>
  );
}

export default App;
