import ThresholdEditor from "../components/ThresholdEditor";
import type { Machine } from "../types";
import type { MachineOption, ThresholdFormRow } from "../lib/app-types";

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
  return (
    <div className="grid two-columns">
      <section className="panel">
        <h3 className="panel-title">全局阈值</h3>
        <form onSubmit={props.onSaveGlobalThresholds}>
          <ThresholdEditor rows={props.globalThresholdForm} onChange={props.onChangeGlobalThresholdForm} />
          <button className="primary-button" disabled={props.busy || props.globalThresholdsSaved} type="submit">
            保存全局阈值
          </button>
        </form>
      </section>

      <section className="panel">
        <div className="panel-header-inline">
          <h3 className="panel-title">单机覆盖阈值</h3>
          <select
            value={props.selectedMachineID ?? ""}
            onChange={(event) => props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
          >
            <option value="">请选择机器</option>
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
              保存 {props.selectedMachine.name} ({props.selectedMachine.host}) 的阈值
            </button>
          </form>
        ) : (
          <p className="muted">请先选择机器。</p>
        )}
      </section>
    </div>
  );
}
