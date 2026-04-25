import { useState } from "react";
import type { FormEvent } from "react";
import ThresholdEditor from "../components/ThresholdEditor";
import type { Machine } from "../types";
import type { MachineOption, ThresholdFormRow } from "../lib/app-types";
import { useI18n } from "../lib/i18n";

type ThresholdsPageProps = {
  busy: boolean;
  globalThresholdForm: ThresholdFormRow[];
  machineThresholdForm: ThresholdFormRow[];
  globalThresholdsSaved: boolean;
  machineThresholdsSaved: boolean;
  selectedMachineID: number | null;
  selectedMachine: Machine | null;
  machineOptions: MachineOption[];
  onSelectMachine: (machineID: number | null) => void;
  onChangeGlobalThresholdForm: (rows: ThresholdFormRow[]) => void;
  onChangeMachineThresholdForm: (rows: ThresholdFormRow[]) => void;
  onSaveGlobalThresholds: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
  onSaveMachineThresholds: (event: FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export default function ThresholdsPage(props: ThresholdsPageProps) {
  const { t } = useI18n();
  const [isMachineModalOpen, setMachineModalOpen] = useState(false);

  function handleSelectMachine(machineID: number | null) {
    props.onSelectMachine(machineID);
    if (!machineID) {
      setMachineModalOpen(false);
    }
  }

  return (
    <div className="page-stack">
      <section className="panel section-panel list-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <p className="section-kicker">{t("thresholdsGlobalTitle")}</p>
              <h3 className="panel-title">{t("thresholdsGlobalTitle")}</h3>
            </div>
            <p className="section-description">{t("thresholdsGlobalDescription")}</p>
          </div>
          <div className="action-row threshold-toolbar-actions">
            <select
              className="threshold-machine-select"
              value={props.selectedMachineID ?? ""}
              onChange={(event) => handleSelectMachine(event.target.value ? Number(event.target.value) : null)}
            >
              <option value="">{t("thresholdsSelectMachine")}</option>
              {props.machineOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <button
              className="secondary-button"
              disabled={props.busy || !props.selectedMachine}
              onClick={() => setMachineModalOpen(true)}
              type="button"
            >
              {t("thresholdsEditMachine")}
            </button>
            <button
              className="primary-button"
              disabled={props.busy || props.globalThresholdsSaved}
              form="global-threshold-form"
              type="submit"
            >
              {t("thresholdsSaveGlobal")}
            </button>
          </div>
        </div>

        <form id="global-threshold-form" onSubmit={props.onSaveGlobalThresholds}>
          <ThresholdEditor
            rows={props.globalThresholdForm}
            onChange={props.onChangeGlobalThresholdForm}
            showSource={false}
          />
        </form>
      </section>

      {isMachineModalOpen && props.selectedMachine ? (
        <div className="modal-backdrop" role="presentation">
          <section className="modal-panel threshold-modal-panel" aria-modal="true" role="dialog">
            <div className="modal-header">
              <div>
                <p className="section-kicker">{t("thresholdsMachineTitle")}</p>
                <h3 className="panel-title">{props.selectedMachine.name}</h3>
                <p className="section-description">{props.selectedMachine.host}</p>
              </div>
              <button className="secondary-button modal-close-button" onClick={() => setMachineModalOpen(false)} type="button">
                {t("cancel")}
              </button>
            </div>

            <form className="form-grid" onSubmit={props.onSaveMachineThresholds}>
              <ThresholdEditor rows={props.machineThresholdForm} onChange={props.onChangeMachineThresholdForm} />
              <div className="modal-actions">
                <button className="secondary-button" onClick={() => setMachineModalOpen(false)} type="button">
                  {t("cancel")}
                </button>
                <button className="primary-button" disabled={props.busy || props.machineThresholdsSaved} type="submit">
                  {t("thresholdsSaveMachine", { name: props.selectedMachine.name, host: props.selectedMachine.host })}
                </button>
              </div>
            </form>
          </section>
        </div>
      ) : null}
    </div>
  );
}
