import { useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { del, get, patch, post, put, withQuery } from "./api";
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
  BackupImportResponse,
  CleanupHistoryResponse,
  CollectNowResponse,
  ConnectionTestResponse,
  EncryptedBackup,
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

const emptyBackupExportForm = () => ({
  password: "",
  includeAllMachines: true,
  machineIDs: [] as number[],
  includeAllSSHKeys: true,
  sshKeyIDs: [] as number[],
});

const defaultListPageSize = 50;

function AppIcon() {
  return (
    <span className="app-icon" aria-hidden="true">
      <svg viewBox="0 0 32 32" focusable="false">
        <path d="M7 21L12 15.5L17 18.5L25 9" />
        <circle cx="7" cy="21" r="1.7" />
        <circle cx="25" cy="9" r="1.7" />
        <path d="M7 25H25" className="app-icon-base" />
      </svg>
    </span>
  );
}

type ProtectedDataLoadOptions = {
  samplesPage?: number;
  sampleMachineID?: number | null;
  samplePeriodType?: string;
  samplesPageSize?: number;
  alertsPage?: number;
  alertMachineID?: number | null;
  alertsPageSize?: number;
};

function App() {
  const { language, setLanguage, t } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const activeTab = tabKeyFromPath(location.pathname);

  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [profile, setProfile] = useState<AdminProfile | null>(null);
  const [isActionMenuOpen, setActionMenuOpen] = useState(false);
  const [isLanguageMenuOpen, setLanguageMenuOpen] = useState(false);
  const [isAccountMenuOpen, setAccountMenuOpen] = useState(false);
  const [backupModalMode, setBackupModalMode] = useState<"export" | "import" | null>(null);
  const [backupExportForm, setBackupExportForm] = useState(emptyBackupExportForm);
  const [backupImportPassword, setBackupImportPassword] = useState("");
  const [backupImportFileName, setBackupImportFileName] = useState("");
  const [backupImportFile, setBackupImportFile] = useState<EncryptedBackup | null>(null);
  const adminInitials = profile?.username.slice(0, 2).toUpperCase() || "AD";
  const currentLanguageLabel = language === "zh" ? t("languageChinese") : t("languageEnglish");
  const currentLanguageBadge = language === "zh" ? "中" : "EN";

  const isSSHKeyMismatchError =
    error.includes("APP_MASTER_KEY") || error.includes("SSH 私钥无法解密") || error.includes("ssh key decrypt failed");

  const [loginForm, setLoginForm] = useState<LoginFormState>({
    username: "admin",
    password: "",
  });

  const [sshKeys, setSSHKeys] = useState<SSHKey[]>([]);
  const [machines, setMachines] = useState<Machine[]>([]);
  const [notificationChannels, setNotificationChannels] = useState<NotificationChannel[]>([]);
  const [samples, setSamples] = useState<TrafficSample[]>([]);
  const [samplesTotal, setSamplesTotal] = useState(0);
  const [samplesPage, setSamplesPage] = useState(1);
  const [samplesPageSize, setSamplesPageSize] = useState(defaultListPageSize);
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [alertsTotal, setAlertsTotal] = useState(0);
  const [alertsPage, setAlertsPage] = useState(1);
  const [alertsPageSize, setAlertsPageSize] = useState(defaultListPageSize);

  const [selectedThresholdMachineID, setSelectedThresholdMachineID] = useState<number | null>(null);
  const [selectedSampleMachineID, setSelectedSampleMachineID] = useState<number | null>(null);
  const [selectedSamplePeriodType, setSelectedSamplePeriodType] = useState("");
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
  const connectionResultTimers = useRef<Record<number, number>>({});
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

  useEffect(() => () => {
    Object.values(connectionResultTimers.current).forEach(window.clearTimeout);
  }, []);

  const selectedMachine = useMemo(
    () => machines.find((machine) => machine.id === selectedThresholdMachineID) ?? null,
    [machines, selectedThresholdMachineID],
  );
  const enabledMachineCount = useMemo(
    () => machines.filter((machine) => machine.collect_enabled).length,
    [machines],
  );
  const activeNotificationCount = useMemo(
    () => notificationChannels.filter((channel) => channel.enabled).length,
    [notificationChannels],
  );
  const pageDescription = useMemo(() => {
    switch (activeTab) {
      case "overview":
        return t("dashboardTagline");
      case "machines":
        return t("machinesPageDescription");
      case "sshKeys":
        return t("sshKeysPageDescription");
      case "thresholds":
        return t("thresholdsPageDescription");
      case "notifications":
        return t("notificationsPageDescription");
      case "samples":
        return t("samplesPageDescription");
      case "alerts":
        return t("alertsPageDescription");
      default:
        return t("dashboardDescription");
    }
  }, [activeTab, t]);

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    setActionMenuOpen(false);
    setAccountMenuOpen(false);
    setLanguageMenuOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    if (!profile) {
      return;
    }
    void loadProtectedData();
  }, [profile]);

  useEffect(() => {
    setMachineThresholdsSaved(true);
    if (!selectedThresholdMachineID || !profile) {
      setMachineThresholdForm(emptyThresholdRows());
      return;
    }
    void loadMachineThresholds(selectedThresholdMachineID);
  }, [selectedThresholdMachineID, profile]);

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
      setGlobalThresholdForm(toThresholdFormRows(globalRules));
    } catch (loadError) {
      setError(toErrorMessage(loadError, language));
    }
  }

  function trafficSamplesPath(page: number, machineID: number | null, pageSize = samplesPageSize, periodType = selectedSamplePeriodType) {
    return withQuery("/api/v1/traffic-samples", {
      machine_id: machineID,
      period_type: periodType,
      page,
      page_size: pageSize,
    });
  }

  function alertsPath(page: number, machineID: number | null, pageSize = alertsPageSize) {
    return withQuery("/api/v1/alerts", {
      machine_id: machineID,
      page,
      page_size: pageSize,
    });
  }

  async function loadSamplesPage(page = samplesPage, machineID = selectedSampleMachineID, pageSize = samplesPageSize, periodType = selectedSamplePeriodType) {
    setBusy(true);
    setError("");
    try {
      const response = await get<TrafficSampleList>(trafficSamplesPath(page, machineID, pageSize, periodType));
      setSamples(response.items);
      setSamplesTotal(response.total);
    } catch (loadError) {
      setError(toErrorMessage(loadError, language));
    } finally {
      setBusy(false);
    }
  }

  async function loadAlertsPage(page = alertsPage, machineID = selectedAlertMachineID, pageSize = alertsPageSize) {
    setBusy(true);
    setError("");
    try {
      const response = await get<AlertList>(alertsPath(page, machineID, pageSize));
      setAlerts(response.items);
      setAlertsTotal(response.total);
    } catch (loadError) {
      setError(toErrorMessage(loadError, language));
    } finally {
      setBusy(false);
    }
  }

  async function loadProtectedData(options: ProtectedDataLoadOptions = {}) {
    const nextSamplesPage = options.samplesPage ?? samplesPage;
    const nextSampleMachineID =
      options.sampleMachineID !== undefined ? options.sampleMachineID : selectedSampleMachineID;
    const nextSamplePeriodType =
      options.samplePeriodType !== undefined ? options.samplePeriodType : selectedSamplePeriodType;
    const nextSamplesPageSize = options.samplesPageSize ?? samplesPageSize;
    const nextAlertsPage = options.alertsPage ?? alertsPage;
    const nextAlertMachineID = options.alertMachineID !== undefined ? options.alertMachineID : selectedAlertMachineID;
    const nextAlertsPageSize = options.alertsPageSize ?? alertsPageSize;

    setBusy(true);
    setError("");
    try {
      const [sshKeysResp, machinesResp, globalRules, channelsResp, samplesResp, alertsResp] = await Promise.all([
        get<SSHKey[]>("/api/v1/ssh-keys"),
        get<Machine[]>("/api/v1/machines"),
        get<ThresholdRule[]>("/api/v1/thresholds/global"),
        get<NotificationChannel[]>("/api/v1/notification-channels"),
        get<TrafficSampleList>(trafficSamplesPath(nextSamplesPage, nextSampleMachineID, nextSamplesPageSize, nextSamplePeriodType)),
        get<AlertList>(alertsPath(nextAlertsPage, nextAlertMachineID, nextAlertsPageSize)),
      ]);

      setSSHKeys(sshKeysResp);
      setMachines(machinesResp);
      setGlobalThresholdForm(toThresholdFormRows(globalRules));
      setNotificationChannels(channelsResp);
      syncChannelForms(channelsResp);
      setSamples(samplesResp.items);
      setSamplesTotal(samplesResp.total);
      setAlerts(alertsResp.items);
      setAlertsTotal(alertsResp.total);

      setSelectedThresholdMachineID((currentSelectedMachineID) => {
        if (currentSelectedMachineID === null) {
          return null;
        }

        return machinesResp.some((machine) => machine.id === currentSelectedMachineID)
          ? currentSelectedMachineID
          : machinesResp[0]?.id ?? null;
      });
      setSelectedSampleMachineID((currentSelectedMachineID) =>
        currentSelectedMachineID && !machinesResp.some((machine) => machine.id === currentSelectedMachineID)
          ? null
          : currentSelectedMachineID,
      );
      setSelectedAlertMachineID((currentSelectedMachineID) =>
        currentSelectedMachineID && !machinesResp.some((machine) => machine.id === currentSelectedMachineID)
          ? null
          : currentSelectedMachineID,
      );
    } catch (loadError) {
      setError(toErrorMessage(loadError, language));
    } finally {
      setBusy(false);
    }
  }

  async function loadMachineThresholds(machineID: number) {
    try {
      const rules = await get<ThresholdRule[]>(`/api/v1/machines/${machineID}/thresholds`);
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

  function handleLanguageSelect(nextLanguage: "zh" | "en") {
    setLanguage(nextLanguage);
    setLanguageMenuOpen(false);
  }

  function openBackupModal(mode: "export" | "import") {
    setActionMenuOpen(false);
    setLanguageMenuOpen(false);
    setAccountMenuOpen(false);
    setBackupModalMode(mode);
  }

  function closeBackupModal() {
    setBackupModalMode(null);
    setBackupImportPassword("");
    setBackupImportFileName("");
    setBackupImportFile(null);
  }

  function selectedNumberOptions(event: ChangeEvent<HTMLSelectElement>) {
    return Array.from(event.currentTarget.selectedOptions).map((option) => Number(option.value));
  }

  function backupFileName() {
    const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
    return `traffic-monitor-backup-${timestamp}.json`;
  }

  function downloadBackupFile(backup: EncryptedBackup) {
    const blob = new Blob([JSON.stringify(backup, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = backupFileName();
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }

  async function handleBackupFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0] ?? null;
    setBackupImportFile(null);
    setBackupImportFileName(file?.name ?? "");

    if (!file) {
      return;
    }

    try {
      const content = await file.text();
      setBackupImportFile(JSON.parse(content) as EncryptedBackup);
    } catch {
      setError(t("backupInvalidFile"));
    }
  }

  async function handleBackupExportSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      const backup = await post<
        EncryptedBackup,
        {
          password: string;
          include_all_machines: boolean;
          machine_ids: number[];
          include_all_ssh_keys: boolean;
          ssh_key_ids: number[];
        }
      >("/api/v1/backups/export", {
        password: backupExportForm.password,
        include_all_machines: backupExportForm.includeAllMachines,
        machine_ids: backupExportForm.machineIDs,
        include_all_ssh_keys: backupExportForm.includeAllSSHKeys,
        ssh_key_ids: backupExportForm.sshKeyIDs,
      });

      downloadBackupFile(backup);
      setBackupExportForm(emptyBackupExportForm());
      setBackupModalMode(null);
      setToast(t("backupExportSuccess"));
    });
  }

  async function handleBackupImportSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!backupImportFile) {
      setError(t("backupFileRequired"));
      return;
    }

    await submitAction(async () => {
      const response = await post<BackupImportResponse, { password: string; backup: EncryptedBackup }>(
        "/api/v1/backups/import",
        {
          password: backupImportPassword,
          backup: backupImportFile,
        },
      );
      await loadProtectedData();
      closeBackupModal();
      setToast(t("backupImportSuccess", {
        sshKeys: response.imported_ssh_keys,
        skippedSSHKeys: response.skipped_ssh_keys,
        machines: response.imported_machines,
        skippedMachines: response.skipped_machines,
      }));
    });
  }

  function renderBackupModal() {
    if (!backupModalMode) {
      return null;
    }

    return (
      <div className="modal-backdrop" role="presentation">
        <section className="modal-panel backup-modal-panel" aria-modal="true" role="dialog">
          <div className="modal-header">
            <div>
              <h3 className="panel-title">
                {backupModalMode === "export" ? t("backupExportTitle") : t("backupImportTitle")}
              </h3>
            </div>
            <button className="secondary-button modal-close-button" onClick={closeBackupModal} type="button">
              {t("cancel")}
            </button>
          </div>

          {backupModalMode === "export" ? (
            <form className="form-grid backup-form-grid" onSubmit={handleBackupExportSubmit}>
              <label className="field full-width">
                <span>{t("backupPassword")}</span>
                <input
                  type="password"
                  required
                  value={backupExportForm.password}
                  onChange={(event) =>
                    setBackupExportForm((current) => ({ ...current, password: event.target.value }))
                  }
                  placeholder={t("backupPasswordPlaceholder")}
                />
              </label>
              <section className="card backup-option-card">
                <label className="checkbox-field">
                  <input
                    checked={backupExportForm.includeAllMachines}
                    onChange={(event) =>
                      setBackupExportForm((current) => ({
                        ...current,
                        includeAllMachines: event.target.checked,
                      }))
                    }
                    type="checkbox"
                  />
                  <span>{t("backupAllMachines")}</span>
                </label>
                <label className="field">
                  <span>{t("backupSelectedMachines")}</span>
                  <select
                    disabled={backupExportForm.includeAllMachines}
                    multiple
                    value={backupExportForm.machineIDs.map(String)}
                    onChange={(event) =>
                      setBackupExportForm((current) => ({
                        ...current,
                        machineIDs: selectedNumberOptions(event),
                      }))
                    }
                  >
                    {machines.map((machine) => (
                      <option key={machine.id} value={machine.id}>
                        {machine.name} ({machine.host})
                      </option>
                    ))}
                  </select>
                </label>
              </section>
              <section className="card backup-option-card">
                <label className="checkbox-field">
                  <input
                    checked={backupExportForm.includeAllSSHKeys}
                    onChange={(event) =>
                      setBackupExportForm((current) => ({
                        ...current,
                        includeAllSSHKeys: event.target.checked,
                      }))
                    }
                    type="checkbox"
                  />
                  <span>{t("backupAllSSHKeys")}</span>
                </label>
                <label className="field">
                  <span>{t("backupSelectedSSHKeys")}</span>
                  <select
                    disabled={backupExportForm.includeAllSSHKeys}
                    multiple
                    value={backupExportForm.sshKeyIDs.map(String)}
                    onChange={(event) =>
                      setBackupExportForm((current) => ({
                        ...current,
                        sshKeyIDs: selectedNumberOptions(event),
                      }))
                    }
                  >
                    {sshKeys.map((sshKey) => (
                      <option key={sshKey.id} value={sshKey.id}>
                        {sshKey.name}
                      </option>
                    ))}
                  </select>
                </label>
              </section>
              <div className="modal-actions">
                <button className="secondary-button" onClick={closeBackupModal} type="button">
                  {t("cancel")}
                </button>
                <button className="primary-button" disabled={busy} type="submit">
                  {t("backupExportSubmit")}
                </button>
              </div>
            </form>
          ) : (
            <form className="form-grid" onSubmit={handleBackupImportSubmit}>
              <label className="field">
                <span>{t("backupPassword")}</span>
                <input
                  type="password"
                  required
                  value={backupImportPassword}
                  onChange={(event) => setBackupImportPassword(event.target.value)}
                  placeholder={t("backupPasswordPlaceholder")}
                />
              </label>
              <label className="field">
                <span>{t("backupFile")}</span>
                <input
                  accept="application/json,.json"
                  onChange={(event) => void handleBackupFileChange(event)}
                  required
                  type="file"
                />
              </label>
              {backupImportFileName ? <p className="card-meta">{backupImportFileName}</p> : null}
              <div className="modal-actions">
                <button className="secondary-button" onClick={closeBackupModal} type="button">
                  {t("cancel")}
                </button>
                <button className="primary-button" disabled={busy} type="submit">
                  {t("backupImportSubmit")}
                </button>
              </div>
            </form>
          )}
        </section>
      </div>
    );
  }

  function renderActionMenu() {
    return (
      <div className="account-menu-wrapper topbar-action-menu-wrapper">
        <button
          className={`secondary-button topbar-action-button${isActionMenuOpen ? " open" : ""}`}
          aria-expanded={isActionMenuOpen}
          aria-haspopup="menu"
          onClick={() => {
            setAccountMenuOpen(false);
            setLanguageMenuOpen(false);
            setActionMenuOpen((current) => !current);
          }}
          type="button"
        >
          {t("topbarActions")}
          <span className="account-menu-caret" aria-hidden="true" />
        </button>
        {isActionMenuOpen ? (
          <div className="account-menu topbar-action-menu" role="menu">
            <button
              className="account-menu-item"
              onClick={() => {
                setActionMenuOpen(false);
                void handleCleanupHistory();
              }}
              role="menuitem"
              type="button"
            >
              {t("cleanupHistory")}
            </button>
            <button
              className="account-menu-item"
              onClick={() => {
                setActionMenuOpen(false);
                openBackupModal("export");
              }}
              role="menuitem"
              type="button"
            >
              {t("backupExportAction")}
            </button>
            <button
              className="account-menu-item"
              onClick={() => {
                setActionMenuOpen(false);
                openBackupModal("import");
              }}
              role="menuitem"
              type="button"
            >
              {t("backupImportAction")}
            </button>
          </div>
        ) : null}
      </div>
    );
  }

  function renderLanguageMenu() {
    return (
      <div className="account-menu-wrapper language-menu-wrapper">
        <button
          className={`account-chip language-chip${isLanguageMenuOpen ? " open" : ""}`}
          aria-expanded={isLanguageMenuOpen}
          aria-haspopup="menu"
          aria-label={t("languageSwitcherLabel")}
          onClick={() => {
            setActionMenuOpen(false);
            setAccountMenuOpen(false);
            setLanguageMenuOpen((current) => !current);
          }}
          type="button"
        >
          <span className="account-avatar">{currentLanguageBadge}</span>
          <span className="account-copy">
            <strong>{t("languageSwitcherLabel")}</strong>
            <small>{currentLanguageLabel}</small>
          </span>
          <span className="account-menu-caret" aria-hidden="true" />
        </button>
        {isLanguageMenuOpen ? (
          <div className="account-menu language-menu" role="menu">
            <div className="account-menu-header">
              <strong>{t("languageSwitcherLabel")}</strong>
              <span>{currentLanguageLabel}</span>
            </div>
            <button
              className={`account-menu-item language-menu-item${language === "zh" ? " active" : ""}`}
              aria-checked={language === "zh"}
              onClick={() => handleLanguageSelect("zh")}
              role="menuitemradio"
              type="button"
            >
              {t("languageChinese")}
            </button>
            <button
              className={`account-menu-item language-menu-item${language === "en" ? " active" : ""}`}
              aria-checked={language === "en"}
              onClick={() => handleLanguageSelect("en")}
              role="menuitemradio"
              type="button"
            >
              {t("languageEnglish")}
            </button>
          </div>
        ) : null}
      </div>
    );
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
      const nextSampleMachineID = selectedSampleMachineID === id ? null : selectedSampleMachineID;
      const nextAlertMachineID = selectedAlertMachineID === id ? null : selectedAlertMachineID;
      const nextSamplesPage = selectedSampleMachineID === id ? 1 : samplesPage;
      const nextAlertsPage = selectedAlertMachineID === id ? 1 : alertsPage;

      await del<null>(`/api/v1/machines/${id}`);
      if (selectedThresholdMachineID === id) {
        setSelectedThresholdMachineID(null);
      }
      if (selectedSampleMachineID === id) {
        setSelectedSampleMachineID(null);
        setSamplesPage(1);
      }
      if (selectedAlertMachineID === id) {
        setSelectedAlertMachineID(null);
        setAlertsPage(1);
      }
      await loadProtectedData({
        sampleMachineID: nextSampleMachineID,
        samplesPage: nextSamplesPage,
        alertMachineID: nextAlertMachineID,
        alertsPage: nextAlertsPage,
      });
      setToast(t("machineDeleteSuccess"));
    });
  }

  async function handleTestConnection(id: number) {
    await submitAction(async () => {
      const result = await post<ConnectionTestResponse>(`/api/v1/machines/${id}/test-connection`);
      const previousTimer = connectionResultTimers.current[id];

      if (previousTimer) {
        window.clearTimeout(previousTimer);
        delete connectionResultTimers.current[id];
      }

      setConnectionResults((current) => ({ ...current, [id]: result }));
      if (result.status.toLowerCase() === "success" || result.status.toLowerCase() === "ok") {
        connectionResultTimers.current[id] = window.setTimeout(() => {
          setConnectionResults((current) => {
            const next = { ...current };
            delete next[id];
            return next;
          });
          delete connectionResultTimers.current[id];
        }, 10000);
      }
      setToast(result.vnstat_version.trim()
        ? t("machineConnectionDoneWithVersion", { id, version: result.vnstat_version.trim() })
        : t("machineConnectionDone", { id }));
    });
  }

  async function handleSaveGlobalThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      await put<null, { rules: ReturnType<typeof toThresholdPayloads> }>("/api/v1/thresholds/global", {
        rules: toThresholdPayloads(globalThresholdForm, language),
      });
      await loadProtectedData();
      setGlobalThresholdsSaved(true);
      setToast(t("globalThresholdSaveSuccess"));
    });
  }

  async function handleSaveMachineThresholds(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedThresholdMachineID) {
      setError(t("selectMachineFirst"));
      return;
    }

    await submitAction(async () => {
      await put<null, { rules: ReturnType<typeof toThresholdPayloads> }>(
        `/api/v1/machines/${selectedThresholdMachineID}/thresholds`,
        {
          rules: toThresholdPayloads(machineThresholdForm, language),
        },
      );
      await loadMachineThresholds(selectedThresholdMachineID);
      setMachineThresholdsSaved(true);
      setToast(t("machineThresholdSaveSuccess"));
    });
  }

  async function handleSaveWebhook(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await submitAction(async () => {
      const parsedHeaders = safeParseHeaders(webhookForm.headersText, language);
      await put<
        null,
        { enabled: boolean; method: "GET" | "POST"; url: string; headers: Record<string, string>; body: string }
      >("/api/v1/notification-channels/webhook", {
        enabled: webhookForm.enabled,
        method: webhookForm.method,
        url: webhookForm.url,
        headers: parsedHeaders,
        body: webhookForm.bodyText,
      });
      await loadProtectedData();
      setWebhookSaved(true);
      setToast(t("webhookSaveSuccess"));
    });
  }

  async function handleTestWebhook() {
    await submitAction(async () => {
      const parsedHeaders = safeParseHeaders(webhookForm.headersText, language);
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
      setSamplesPage(1);
      setAlertsPage(1);
      await loadProtectedData({ samplesPage: 1, alertsPage: 1 });
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

  async function handleSelectSampleMachine(machineID: number | null) {
    setSelectedSampleMachineID(machineID);
    setSamplesPage(1);
    await loadSamplesPage(1, machineID);
  }

  async function handleSelectSamplePeriodType(periodType: string) {
    setSelectedSamplePeriodType(periodType);
    setSamplesPage(1);
    await loadSamplesPage(1, selectedSampleMachineID, samplesPageSize, periodType);
  }

  async function handleSamplesPageChange(page: number) {
    setSamplesPage(page);
    await loadSamplesPage(page, selectedSampleMachineID);
  }

  async function handleSamplesPageSizeChange(pageSize: number) {
    setSamplesPageSize(pageSize);
    setSamplesPage(1);
    await loadSamplesPage(1, selectedSampleMachineID, pageSize);
  }

  async function handleSelectAlertMachine(machineID: number | null) {
    setSelectedAlertMachineID(machineID);
    setAlertsPage(1);
    await loadAlertsPage(1, machineID);
  }

  async function handleAlertsPageChange(page: number) {
    setAlertsPage(page);
    await loadAlertsPage(page, selectedAlertMachineID);
  }

  async function handleAlertsPageSizeChange(pageSize: number) {
    setAlertsPageSize(pageSize);
    setAlertsPage(1);
    await loadAlertsPage(1, selectedAlertMachineID, pageSize);
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
          <div className="app-brand">
            <AppIcon />
            <p className="eyebrow">traffic-monitor</p>
          </div>
          <div className="panel-header-inline auth-header">
            <div className="auth-copy">
              <h1 className="panel-title">{t("loginTitle")}</h1>
              <p className="muted">{t("loginDescription")}</p>
            </div>
            {renderLanguageMenu()}
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
        <section className="brand-panel">
          <NavLink className="brand-copy brand-home-link" to="/overview">
            <div className="app-brand">
              <AppIcon />
              <p className="eyebrow">traffic-monitor</p>
            </div>
            <h1 className="sidebar-title">{t("sidebarTitle")}</h1>
          </NavLink>
          <div className="brand-metrics">
            <div>
              <span>{t("overviewEnabledMachines")}</span>
              <strong>{enabledMachineCount}</strong>
            </div>
            <div>
              <span>{t("overviewNotificationsLabel")}</span>
              <strong>{activeNotificationCount}</strong>
            </div>
          </div>
        </section>

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
      </aside>

      <section className="content">
        <header className="content-hero">
          <div className="content-hero-copy">
            <h2>{tabTitle(activeTab, language)}</h2>
            <p>{pageDescription}</p>
          </div>
          <div className="topbar-right">
            {renderActionMenu()}
            <div className="account-toolbar">
              {renderLanguageMenu()}
              <div className="account-menu-wrapper">
                <button
                  className={`account-chip${isAccountMenuOpen ? " open" : ""}`}
                  aria-expanded={isAccountMenuOpen}
                  aria-haspopup="menu"
                  onClick={() => {
                    setActionMenuOpen(false);
                    setLanguageMenuOpen(false);
                    setAccountMenuOpen((current) => !current);
                  }}
                  type="button"
                >
                  <span className="account-avatar">{adminInitials}</span>
                  <span className="account-copy">
                    <strong>{profile.username}</strong>
                    <small>Admin</small>
                  </span>
                  <span className="account-menu-caret" aria-hidden="true" />
                </button>
                {isAccountMenuOpen ? (
                  <div className="account-menu" role="menu">
                    <div className="account-menu-header">
                      <strong>{profile.username}</strong>
                      <span>Admin</span>
                    </div>
                    <button
                      className="account-menu-item danger"
                      onClick={() => {
                        setAccountMenuOpen(false);
                        void handleLogout();
                      }}
                      role="menuitem"
                      type="button"
                    >
                      {t("logout")}
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </header>

        {toast ? (
          <div className="message success elevated" role="status" aria-live="polite">
            {toast}
          </div>
        ) : null}
        {error ? (
          <div className={`message elevated ${isSSHKeyMismatchError ? "warning" : "error"}`} role="alert">
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
        {busy ? (
          <div className="message info elevated" role="status" aria-live="polite">
            {t("processingRequest")}
          </div>
        ) : null}

        <Routes>
          <Route path="/" element={<Navigate replace to="/overview" />} />
          <Route
            path="/overview"
            element={
              <OverviewTab
                sshKeys={sshKeys}
                machines={machines}
                notificationChannels={notificationChannels}
                samplesTotal={samplesTotal}
                alertsTotal={alertsTotal}
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
                selectedMachineID={selectedThresholdMachineID}
                selectedMachine={selectedMachine}
                machineOptions={machineOptions}
                onSelectMachine={setSelectedThresholdMachineID}
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
                busy={busy}
                selectedMachineID={selectedSampleMachineID}
                selectedPeriodType={selectedSamplePeriodType}
                machineOptions={machineOptions}
                samples={samples}
                total={samplesTotal}
                page={samplesPage}
                pageSize={samplesPageSize}
                collectResults={collectResults}
                onSelectMachine={(machineID) => void handleSelectSampleMachine(machineID)}
                onSelectPeriodType={(periodType) => void handleSelectSamplePeriodType(periodType)}
                onPageChange={(page) => void handleSamplesPageChange(page)}
                onPageSizeChange={(pageSize) => void handleSamplesPageSizeChange(pageSize)}
                onRefresh={() => void loadSamplesPage()}
                onCollectAllMachines={() => void handleCollectNow()}
                onCollectCurrentMachine={(machineID) => void handleCollectNow(machineID)}
              />
            }
          />
          <Route
            path="/alerts"
            element={
              <AlertsPage
                busy={busy}
                alerts={alerts}
                total={alertsTotal}
                page={alertsPage}
                pageSize={alertsPageSize}
                machineOptions={machineOptions}
                selectedMachineID={selectedAlertMachineID}
                onSelectMachine={(machineID) => void handleSelectAlertMachine(machineID)}
                onPageChange={(page) => void handleAlertsPageChange(page)}
                onPageSizeChange={(pageSize) => void handleAlertsPageSizeChange(pageSize)}
              />
            }
          />
          <Route path="*" element={<Navigate replace to="/overview" />} />
        </Routes>
        {renderBackupModal()}
      </section>
    </main>
  );
}

export default App;
