import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import type { ConnectionTestResponse, Machine, SSHKey } from "../types";
import type { MachineFormState } from "../lib/app-types";
import { formatStatusText } from "../lib/app-utils";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";

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

export default function MachinesPage(props: MachinesPageProps) {
  const { language, t } = useI18n();
  const [isMachineModalOpen, setMachineModalOpen] = useState(false);
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

  useEffect(() => {
    if (editingMachineID) {
      setMachineModalOpen(true);
    }
  }, [editingMachineID]);

  function openCreateMachineModal() {
    onResetMachineForm();
    setMachineModalOpen(true);
  }

  function closeMachineModal() {
    setMachineModalOpen(false);
    onResetMachineForm();
  }

  function startEditMachine(machine: Machine) {
    onStartEditMachine(machine);
    setMachineModalOpen(true);
  }

  async function handleMachineModalSubmit(event: FormEvent<HTMLFormElement>) {
    await onMachineSubmit(event);
    setMachineModalOpen(false);
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
              <p className="section-kicker">{t("machinesInventoryTitle")}</p>
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
          <div className="table-wrapper">
            <table>
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
                {machines.map((machine) => (
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
                    <td>
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
                      {connectionResults[machine.id] ? (
                        <p className="card-meta">
                          {t("machinesTestResult", {
                            status: formatStatusText(connectionResults[machine.id].status, language),
                          })}
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
        )}
      </section>

      {isMachineModalOpen ? (
        <div className="modal-backdrop" role="presentation">
          <section className="modal-panel" aria-modal="true" role="dialog">
            <div className="modal-header">
              <div>
                <p className="section-kicker">{editingMachineID ? t("machinesEditTitle") : t("machinesCreateTitle")}</p>
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

              <label className="field">
                <span>{t("machinesNetworkInterface")}</span>
                <input
                  value={machineForm.networkInterface}
                  onChange={(event) => onUpdateMachineForm("networkInterface", event.target.value)}
                />
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
