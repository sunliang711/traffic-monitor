import { useEffect, useState } from "react";
import type { FormEvent } from "react";
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
  sshKeys: SSHKey[];
  machines: Machine[];
  connectionResults: Record<number, ConnectionTestResponse>;
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
  const [networkInterfaceSelectValue, setNetworkInterfaceSelectValue] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const enabledMachines = props.machines.filter((machine) => machine.collect_enabled).length;
  const {
    busy,
    editingMachineID,
    machineForm,
    machineFormSaved,
    sshKeys,
    machines,
    connectionResults,
    onMachineSubmit,
    onResetMachineForm,
    onUpdateMachineForm,
    onStartEditMachine,
    onTestConnection,
    onDeleteMachine,
  } = props;
  const totalPages = Math.max(1, Math.ceil(machines.length / pageSize));
  const visibleMachines = machines.slice((page - 1) * pageSize, page * pageSize);

  useEffect(() => {
    if (editingMachineID) {
      setNetworkInterfaceSelectValue(networkInterfaceMode(machineForm.networkInterface));
      setMachineModalOpen(true);
    }
  }, [editingMachineID]);

  useEffect(() => {
    setPage((current) => Math.min(current, totalPages));
  }, [totalPages]);

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
          <button className="primary-button" onClick={openCreateMachineModal} type="button">
            {t("machinesCreate")}
          </button>
        </div>

        {machines.length === 0 ? (
          <EmptyState
            title={t("machinesEmptyTitle")}
            description={t("machinesEmptyDescription")}
            action={
              <button className="primary-button" onClick={openCreateMachineModal} type="button">
                {t("machinesCreate")}
              </button>
            }
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
                    <th>{t("machinesActions")}</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleMachines.map((machine) => {
                    const connectionResult = connectionResults[machine.id];
                    const connectionStatus = connectionResult?.status.toLowerCase();
                    const connectionResultType = connectionStatus === "success" || connectionStatus === "ok" ? "ok" : "error";

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
                        <td className="machine-actions-cell">
                          <div className="machine-actions-stack">
                            <div className="action-row">
                              <button className="secondary-button" onClick={() => startEditMachine(machine)} type="button">
                                {t("machinesEdit")}
                              </button>
                              <button className="secondary-button" onClick={() => void onTestConnection(machine.id)} type="button">
                                {t("machinesTest")}
                              </button>
                              <button className="danger-button" onClick={() => void onDeleteMachine(machine.id)} type="button">
                                {t("machinesDelete")}
                              </button>
                            </div>
                            {connectionResult ? (
                              connectionResultType === "ok" ? (
                                <span
                                  className="machine-test-result ok machine-test-success-icon"
                                  title={connectionResult.vnstat_version || undefined}
                                  aria-label={t("machinesTestResult", {
                                    status: formatStatusText(connectionResult.status, language),
                                  })}
                                >
                                  ✓
                                </span>
                              ) : (
                                <span
                                  className={`machine-test-result ${connectionResultType}`}
                                  title={connectionResult.vnstat_version || undefined}
                                >
                                  <span className="machine-test-dot" aria-hidden="true" />
                                  <span className="machine-test-label">
                                    {t("machinesTestResult", {
                                      status: formatStatusText(connectionResult.status, language),
                                    })}
                                  </span>
                                  {connectionResult.vnstat_version ? (
                                    <span className="machine-test-version">{shortVnstatVersion(connectionResult.vnstat_version)}</span>
                                  ) : null}
                                </span>
                              )
                            ) : null}
                          </div>
                        </td>
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
                {t("cancel")}
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
    </div>
  );
}
