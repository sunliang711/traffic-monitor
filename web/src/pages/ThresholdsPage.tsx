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
  const isMachineThresholdSelected = props.selectedMachineID !== null && props.selectedMachine !== null;
  const activeRows = isMachineThresholdSelected
    ? props.machineThresholdForm
    : props.globalThresholdForm.map((row) => ({ ...row, source: row.source ?? "global" }));
  const activeDescription = isMachineThresholdSelected ? t("thresholdsMachineDescription") : t("thresholdsGlobalDescription");
  const activeSaveDisabled = isMachineThresholdSelected ? props.machineThresholdsSaved : props.globalThresholdsSaved;
  const activeSubmitHandler = isMachineThresholdSelected ? props.onSaveMachineThresholds : props.onSaveGlobalThresholds;
  const activeChangeHandler = isMachineThresholdSelected ? props.onChangeMachineThresholdForm : props.onChangeGlobalThresholdForm;

  function handleSelectMachine(machineID: number | null) {
    props.onSelectMachine(machineID);
  }

  return (
    <div className="page-stack">
      <section className="panel section-panel list-panel">
        <div className="section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">
                {isMachineThresholdSelected ? props.selectedMachine?.name : t("thresholdsGlobalTitle")}
              </h3>
            </div>
            <p className="section-description">{activeDescription}</p>
          </div>
          <div className="action-row threshold-toolbar-actions">
            <select
              className="threshold-machine-select"
              value={props.selectedMachineID ?? ""}
              onChange={(event) => handleSelectMachine(event.target.value ? Number(event.target.value) : null)}
            >
              <option value="">{t("thresholdsGlobalTitle")}</option>
              {props.machineOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <button
              className="primary-button"
              disabled={props.busy || activeSaveDisabled}
              form="threshold-form"
              type="submit"
            >
              {t("save")}
            </button>
          </div>
        </div>

        <form id="threshold-form" onSubmit={activeSubmitHandler}>
          <ThresholdEditor
            rows={activeRows}
            onChange={activeChangeHandler}
          />
        </form>
      </section>
    </div>
  );
}
