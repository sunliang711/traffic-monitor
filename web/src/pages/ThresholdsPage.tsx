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
  onSaveGlobalThresholds: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
  onSaveMachineThresholds: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export default function ThresholdsPage(props: ThresholdsPageProps) {
  const { t } = useI18n();

  return (
    <div className="page-stack">
      <div className="grid dashboard-columns">
      <section className="panel section-panel">
        <div className="section-intro">
          <div>
            <p className="section-kicker">{t("thresholdsGlobalTitle")}</p>
            <h3 className="panel-title">{t("thresholdsGlobalTitle")}</h3>
          </div>
          <p className="section-description">{t("thresholdsGlobalDescription")}</p>
        </div>
        <form onSubmit={props.onSaveGlobalThresholds}>
          <ThresholdEditor rows={props.globalThresholdForm} onChange={props.onChangeGlobalThresholdForm} />
          <button className="primary-button" disabled={props.busy || props.globalThresholdsSaved} type="submit">
            {t("thresholdsSaveGlobal")}
          </button>
        </form>
      </section>

      <section className="panel section-panel">
        <div className="section-intro">
          <div>
            <p className="section-kicker">{t("thresholdsMachineTitle")}</p>
            <h3 className="panel-title">{t("thresholdsMachineTitle")}</h3>
          </div>
          <p className="section-description">{t("thresholdsMachineDescription")}</p>
        </div>
        <div className="panel-header-inline">
          <select
            value={props.selectedMachineID ?? ""}
            onChange={(event) => props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
          >
            <option value="">{t("thresholdsSelectMachine")}</option>
            {props.machineOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        {props.selectedMachine ? (
          <form onSubmit={props.onSaveMachineThresholds}>
            <ThresholdEditor rows={props.machineThresholdForm} onChange={props.onChangeMachineThresholdForm} />
            <button className="primary-button" disabled={props.busy || props.machineThresholdsSaved} type="submit">
              {t("thresholdsSaveMachine", { name: props.selectedMachine.name, host: props.selectedMachine.host })}
            </button>
          </form>
        ) : (
          <p className="muted">{t("thresholdsSelectMachineFirst")}</p>
        )}
      </section>
      </div>
    </div>
  );
}
