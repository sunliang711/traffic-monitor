import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import { createPortal } from "react-dom";
import type { ConnectionTestResponse, Machine, SSHKey } from "../types";
import type { MachineFormState } from "../lib/app-types";
import { formatStatusText } from "../lib/app-utils";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";
import PageSizeSelect from "../components/PageSizeSelect";

type MachinesPageProps = {
  busy: boolean;
  editingMachineID: number | null;
  machineForm: MachineFormState;
  machineFormSaved: boolean;
  readOnly?: boolean;
  sshKeys: SSHKey[];
  machines: Machine[];
  connectionResults: Record<number, ConnectionTestResponse>;
  testingMachineIDs: Record<number, boolean>;
  onMachineSubmit: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onResetMachineForm: () => void;
  onUpdateMachineForm: <Key extends keyof MachineFormState>(key: Key, value: MachineFormState[Key]) => void;
  onStartEditMachine: (machine: Machine) => void;
  onTestConnection: (id: number) => void | Promise<void>;
  onDeleteMachine: (id: number) => void | Promise<void>;
};

function sshKeyName(sshKeys: SSHKey[], sshKeyID: number) {
  return sshKeys.find((sshKey) => sshKey.id === sshKeyID)?.name ?? `SSH Key ${sshKeyID}`;
}

function shortVnstatVersion(version: string) {
  const matched = version.trim().match(/^(vnStat\s+\S+)/i);

  return matched?.[1] ?? version;
}

function formatTestDuration(elapsedMS?: number) {
  if (typeof elapsedMS !== "number") {
    return "";
  }

  const safeElapsedMS = Math.max(0, Math.round(elapsedMS));
  if (safeElapsedMS < 1000) {
    return `${safeElapsedMS}ms`;
  }

  if (safeElapsedMS < 10000) {
    return `${(safeElapsedMS / 1000).toFixed(3)}s`;
  }

  return `${(safeElapsedMS / 1000).toFixed(1)}s`;
}

function formatTestDurationPair(backendElapsedMS?: number, frontendElapsedMS?: number) {
  const backendDuration = formatTestDuration(backendElapsedMS);
  const frontendDuration = formatTestDuration(frontendElapsedMS);
  if (!backendDuration && !frontendDuration) {
    return "";
  }

  return `${backendDuration || "-"}/${frontendDuration || "-"}`;
}

const suggestedNetworkInterfaces = ["eth0", "ens18"];
const customNetworkInterfaceValue = "custom";

function networkInterfaceMode(value: string) {
  if (!value) {
    return "";
  }

  return suggestedNetworkInterfaces.includes(value) ? value : customNetworkInterfaceValue;
}

export default function MachinesPage(props: MachinesPageProps) {
  const { language, t } = useI18n();
  const [isMachineModalOpen, setMachineModalOpen] = useState(false);
  const [testPopoverMachineID, setTestPopoverMachineID] = useState<number | null>(null);
  const [testPopoverPosition, setTestPopoverPosition] = useState({ top: 0, left: 0 });
  const [networkInterfaceSelectValue, setNetworkInterfaceSelectValue] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const testButtonRefs = useRef<Record<number, HTMLButtonElement | null>>({});
  const enabledMachines = props.machines.filter((machine) => machine.collect_enabled).length;
  const {
    busy,
    editingMachineID,
    machineForm,
    machineFormSaved,
    readOnly,
    sshKeys,
    machines,
    connectionResults,
    testingMachineIDs,
    onMachineSubmit,
    onResetMachineForm,
    onUpdateMachineForm,
    onStartEditMachine,
    onTestConnection,
    onDeleteMachine,
  } = props;
  const totalPages = Math.max(1, Math.ceil(machines.length / pageSize));
  const visibleMachines = machines.slice((page - 1) * pageSize, page * pageSize);
  const testPopoverMachine = testPopoverMachineID
    ? machines.find((machine) => machine.id === testPopoverMachineID) ?? null
    : null;
  const testPopoverResult = testPopoverMachineID ? connectionResults[testPopoverMachineID] : undefined;
  const isTestPopoverPending = testPopoverMachineID ? Boolean(testingMachineIDs[testPopoverMachineID]) : false;
  const testPopoverDuration = formatTestDurationPair(testPopoverResult?.backend_elapsed_ms, testPopoverResult?.frontend_elapsed_ms);
  const testPopoverStatusText = isTestPopoverPending
    ? t("machinesTesting")
    : testPopoverResult
      ? formatStatusText(testPopoverResult.status, language)
      : t("machinesTestNotRun");
  const testPopoverStatusClass = isTestPopoverPending || !testPopoverResult ? "idle" : testPopoverResult.status === "ok" ? "ok" : "error";
  const testPopoverMeta = [
    testPopoverResult?.vnstat_version ? shortVnstatVersion(testPopoverResult.vnstat_version) : "",
    testPopoverDuration ? t("machinesTestDuration", { value: testPopoverDuration }) : "",
  ]
    .filter(Boolean)
    .join(" · ");

  useEffect(() => {
    if (editingMachineID) {
      setNetworkInterfaceSelectValue(networkInterfaceMode(machineForm.networkInterface));
      setMachineModalOpen(true);
    }
  }, [editingMachineID]);

  useEffect(() => {
    setPage((current) => Math.min(current, totalPages));
  }, [totalPages]);

  useEffect(() => {
    if (!testPopoverMachineID) {
      return;
    }

    if (!visibleMachines.some((machine) => machine.id === testPopoverMachineID)) {
      setTestPopoverMachineID(null);
    }
  }, [testPopoverMachineID, visibleMachines]);

  useEffect(() => {
    function handleDocumentMouseDown(event: MouseEvent) {
      const target = event.target as HTMLElement | null;
      if (target?.closest(".machine-test-popover") || target?.closest(".machine-test-button")) {
        return;
      }

      setTestPopoverMachineID(null);
    }

    function handleDocumentKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setTestPopoverMachineID(null);
      }
    }

    document.addEventListener("mousedown", handleDocumentMouseDown);
    document.addEventListener("keydown", handleDocumentKeyDown);
    return () => {
      document.removeEventListener("mousedown", handleDocumentMouseDown);
      document.removeEventListener("keydown", handleDocumentKeyDown);
    };
  }, []);

  useLayoutEffect(() => {
    function updateTestPopoverPosition() {
      if (!testPopoverMachineID) {
        return;
      }

      const button = testButtonRefs.current[testPopoverMachineID];
      if (!button) {
        return;
      }

      const rect = button.getBoundingClientRect();
      const popoverWidth = Math.min(420, window.innerWidth - 32);
      const popoverHeight = 220;
      const left = Math.min(Math.max(16, rect.right - popoverWidth), window.innerWidth - popoverWidth - 16);
      let top = rect.bottom + 10;
      if (top + popoverHeight > window.innerHeight) {
        top = Math.max(16, rect.top - popoverHeight - 10);
      }

      setTestPopoverPosition({ top, left });
    }

    updateTestPopoverPosition();
    window.addEventListener("resize", updateTestPopoverPosition);
    window.addEventListener("scroll", updateTestPopoverPosition, true);
    return () => {
      window.removeEventListener("resize", updateTestPopoverPosition);
      window.removeEventListener("scroll", updateTestPopoverPosition, true);
    };
  }, [testPopoverMachineID]);

  function openCreateMachineModal() {
    onResetMachineForm();
    setNetworkInterfaceSelectValue("");
    setMachineModalOpen(true);
  }

  function closeMachineModal() {
    setMachineModalOpen(false);
    setNetworkInterfaceSelectValue("");
    onResetMachineForm();
  }

  function startEditMachine(machine: Machine) {
    setNetworkInterfaceSelectValue(networkInterfaceMode(machine.network_interface));
    onStartEditMachine(machine);
    setMachineModalOpen(true);
  }

  async function handleMachineModalSubmit(event: FormEvent<HTMLFormElement>) {
    await onMachineSubmit(event);
    setMachineModalOpen(false);
  }

  function handlePageSizeChange(nextPageSize: number) {
    setPageSize(nextPageSize);
    setPage(1);
  }

  function handleNetworkInterfaceSelect(value: string) {
    setNetworkInterfaceSelectValue(value);

    if (value === customNetworkInterfaceValue) {
      onUpdateMachineForm("networkInterface", "");
      return;
    }

    onUpdateMachineForm("networkInterface", value);
  }

  function handleTestConnection(machineID: number) {
    setTestPopoverMachineID(machineID);
    void onTestConnection(machineID);
  }

  return (
    <div className="page-stack">
      <section className="summary-strip">
        <div className="summary-tile teal compact">
          <span>{t("overviewMachinesLabel")}</span>
          <strong>{machines.length}</strong>
        </div>
        <div className="summary-tile amber compact">
          <span>{t("overviewEnabledMachines")}</span>
          <strong>{enabledMachines}</strong>
        </div>
      </section>

      <section className="panel section-panel list-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("machinesList")}</h3>
            </div>
            <p className="section-description">{t("machinesInventoryDescription")}</p>
          </div>
          {readOnly ? null : (
            <button className="primary-button" onClick={openCreateMachineModal} type="button">
              {t("machinesCreate")}
            </button>
          )}
        </div>

        {machines.length === 0 ? (
          <EmptyState
            title={t("machinesEmptyTitle")}
            description={t("machinesEmptyDescription")}
            action={readOnly ? undefined : (
              <button className="primary-button" onClick={openCreateMachineModal} type="button">
                {t("machinesCreate")}
              </button>
            )}
          />
        ) : (
          <>
            <div className="table-wrapper">
              <table className="machine-table">
                <thead>
                  <tr>
                    <th>{t("machinesName")}</th>
                    <th>{t("machinesHost")}</th>
                    <th>{t("machinesNetworkInterface")}</th>
                    <th>SSH Key</th>
                    <th>{t("machinesCollectEnabled")}</th>
                    {readOnly ? null : <th>{t("machinesActions")}</th>}
                  </tr>
                </thead>
                <tbody>
                  {visibleMachines.map((machine) => {
                    const isTestingConnection = Boolean(testingMachineIDs[machine.id]);

                    return (
                      <tr key={machine.id}>
                        <td>{machine.name}</td>
                        <td>
                          {machine.host}:{machine.port}
                        </td>
                        <td>{machine.network_interface}</td>
                        <td>{sshKeyName(sshKeys, machine.ssh_key_id)}</td>
                        <td>
                          <span className={`status-badge ${machine.collect_enabled ? "ok" : "idle"}`}>
                            {machine.collect_enabled ? t("statusEnabled") : t("statusDisabled")}
                          </span>
                        </td>
                        {readOnly ? null : (
                          <td className="machine-actions-cell">
                            <div className="machine-actions-stack">
                              <div className="action-row">
                                <button className="secondary-button" onClick={() => startEditMachine(machine)} type="button">
                                  {t("machinesEdit")}
                                </button>
                                <button
                                  ref={(element) => {
                                    testButtonRefs.current[machine.id] = element;
                                  }}
                                  className="secondary-button machine-test-button"
                                  disabled={isTestingConnection}
                                  onClick={() => handleTestConnection(machine.id)}
                                  type="button"
                                >
                                  {isTestingConnection ? t("machinesTesting") : t("machinesTest")}
                                </button>
                                <button className="danger-button" onClick={() => void onDeleteMachine(machine.id)} type="button">
                                  {t("machinesDelete")}
                                </button>
                              </div>
                            </div>
                          </td>
                        )}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <div className="pagination-row">
              <div className="pagination-meta">
                <span className="card-meta">{t("samplesPageInfo", { page, totalPages, total: machines.length })}</span>
                <PageSizeSelect value={pageSize} onChange={handlePageSizeChange} />
              </div>
              <div className="action-row">
                <button className="secondary-button" disabled={page <= 1} onClick={() => setPage(page - 1)} type="button">
                  {t("previousPage")}
                </button>
                <button className="secondary-button" disabled={page >= totalPages} onClick={() => setPage(page + 1)} type="button">
                  {t("nextPage")}
                </button>
              </div>
            </div>
          </>
        )}
      </section>

      {isMachineModalOpen ? (
        <div className="modal-backdrop" role="presentation">
          <section className="modal-panel" aria-modal="true" role="dialog">
            <div className="modal-header">
              <div>
                <h3 className="panel-title">{editingMachineID ? t("machinesEditTitle") : t("machinesCreateTitle")}</h3>
              </div>
              <button className="secondary-button modal-close-button" onClick={closeMachineModal} type="button">
                {t("close")}
              </button>
            </div>

            <form className="form-grid machine-form-grid" onSubmit={handleMachineModalSubmit}>
              <label className="field">
                <span>{t("machinesName")}</span>
                <input value={machineForm.name} onChange={(event) => onUpdateMachineForm("name", event.target.value)} />
              </label>

              <label className="field">
                <span>{t("machinesHost")}</span>
                <input value={machineForm.host} onChange={(event) => onUpdateMachineForm("host", event.target.value)} />
              </label>

              <label className="field">
                <span>{t("machinesPort")}</span>
                <input value={machineForm.port} onChange={(event) => onUpdateMachineForm("port", event.target.value)} />
              </label>

              <label className="field">
                <span>{t("machinesSSHUser")}</span>
                <input value={machineForm.sshUser} onChange={(event) => onUpdateMachineForm("sshUser", event.target.value)} />
              </label>

              <label className="field network-interface-field">
                <span>{t("machinesNetworkInterface")}</span>
                <select
                  value={networkInterfaceSelectValue}
                  onChange={(event) => handleNetworkInterfaceSelect(event.target.value)}
                >
                  <option value="">{t("machinesNetworkInterfacePlaceholder")}</option>
                  {suggestedNetworkInterfaces.map((networkInterface) => (
                    <option key={networkInterface} value={networkInterface}>
                      {networkInterface}
                    </option>
                  ))}
                  <option value={customNetworkInterfaceValue}>{t("machinesNetworkInterfaceCustom")}</option>
                </select>
                {networkInterfaceSelectValue === customNetworkInterfaceValue ? (
                  <input
                    value={machineForm.networkInterface}
                    onChange={(event) => onUpdateMachineForm("networkInterface", event.target.value)}
                    placeholder={t("machinesNetworkInterfaceCustomPlaceholder")}
                  />
                ) : null}
              </label>

              <label className="field">
                <span>SSH Key</span>
                <select value={machineForm.sshKeyID} onChange={(event) => onUpdateMachineForm("sshKeyID", event.target.value)}>
                  <option value="">{t("machinesSelectSSHKey")}</option>
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
                  onChange={(event) => onUpdateMachineForm("collectEnabled", event.target.checked)}
                  type="checkbox"
                />
                <span>{t("machinesCollectEnabled")}</span>
              </label>

              <label className="field full-width">
                <span>{t("machinesRemark")}</span>
                <textarea
                  rows={3}
                  value={machineForm.remark}
                  onChange={(event) => onUpdateMachineForm("remark", event.target.value)}
                />
              </label>

              <div className="modal-actions">
                <button className="secondary-button" onClick={closeMachineModal} type="button">
                  {t("cancel")}
                </button>
                <button className="primary-button" disabled={busy || machineFormSaved} type="submit">
                  {editingMachineID ? t("machinesSave") : t("machinesCreate")}
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}

      {testPopoverMachine
        ? createPortal(
            <section
              className="machine-test-popover"
              style={{ top: testPopoverPosition.top, left: testPopoverPosition.left }}
              aria-label={t("machinesTestResult", {
                status: testPopoverStatusText,
              })}
              role="dialog"
            >
              <div className="machine-test-popover-header">
                <div className="machine-test-popover-machine">
                  <strong>{testPopoverMachine.name}</strong>
                  <p className="card-meta">
                    {testPopoverMachine.host}:{testPopoverMachine.port}
                  </p>
                </div>
                <div className="machine-test-popover-result">
                  <span className={`status-badge ${testPopoverStatusClass}`}>{testPopoverStatusText}</span>
                  {testPopoverMeta ? <span className="machine-test-popover-meta">{testPopoverMeta}</span> : null}
                </div>
                <button
                  className="machine-test-popover-close"
                  onClick={() => setTestPopoverMachineID(null)}
                  aria-label={t("close")}
                  type="button"
                >
                  ×
                </button>
              </div>
              {testPopoverResult ? (
                <div className="machine-test-popover-checks">
                  <span className={`machine-test-check ${testPopoverResult.ssh_reachable ? "ok" : "error"}`}>
                    <span aria-hidden="true">{testPopoverResult.ssh_reachable ? "✓" : "×"}</span>
                    {t("machinesCheckSSH")}
                  </span>
                  <span className={`machine-test-check ${testPopoverResult.vnstat_ready ? "ok" : "error"}`}>
                    <span aria-hidden="true">{testPopoverResult.vnstat_ready ? "✓" : "×"}</span>
                    {t("machinesCheckVNStat")}
                  </span>
                  <span className={`machine-test-check ${testPopoverResult.network_interface_ready ? "ok" : "error"}`}>
                    <span aria-hidden="true">{testPopoverResult.network_interface_ready ? "✓" : "×"}</span>
                    {t("machinesCheckInterface")}
                  </span>
                </div>
              ) : null}
            </section>,
            document.body,
          )
        : null}
    </div>
  );
}
