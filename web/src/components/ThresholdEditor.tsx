import { formatMetricType, formatPeriodType, formatThresholdSource } from "../lib/app-utils";
import { useI18n } from "../lib/i18n";

type ThresholdFormRow = {
  period_type: string;
  metric_type: string;
  threshold_value: string;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
  strategy: "inherit" | "override" | "disabled";
  source?: string;
};

type ThresholdEditorProps = {
  rows: ThresholdFormRow[];
  onChange: (rows: ThresholdFormRow[]) => void;
  mode?: "global" | "machine";
  showSource?: boolean;
  readOnly?: boolean;
};

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

export default function ThresholdEditor(props: ThresholdEditorProps) {
  const { language, t } = useI18n();
  const mode = props.mode ?? "global";
  const isMachineMode = mode === "machine";
  const showSource = props.showSource ?? true;

  return (
    <div className={`table-wrapper threshold-editor${isMachineMode ? " machine-threshold-editor" : ""}`}>
      <table>
        <thead>
          <tr>
            <th>{t("thresholdPeriod")}</th>
            <th>{t("thresholdDimension")}</th>
            {isMachineMode ? <th>{t("thresholdStrategy")}</th> : null}
            <th>{t("thresholdValue")}</th>
            <th>{t("thresholdUnit")}</th>
            {isMachineMode ? <th>{t("thresholdEffectiveRule")}</th> : <th>{t("thresholdEnabled")}</th>}
            {!isMachineMode && showSource ? <th>{t("thresholdSource")}</th> : null}
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row, index) => (
            <tr key={`${row.period_type}-${row.metric_type}`}>
              <td>{formatPeriodType(row.period_type, language)}</td>
              <td>{formatMetricType(row.metric_type, language)}</td>
              {isMachineMode ? (
                <td>
                  <select
                    disabled={props.readOnly}
                    value={row.strategy}
                    onChange={(event) =>
                      updateRow(
                        props.rows,
                        index,
                        "strategy",
                        event.target.value as "inherit" | "override" | "disabled",
                        props.onChange,
                      )
                    }
                  >
                    <option value="inherit">{t("thresholdStrategyInherit")}</option>
                    <option value="override">{t("thresholdStrategyOverride")}</option>
                    <option value="disabled">{t("thresholdStrategyDisabled")}</option>
                  </select>
                </td>
              ) : null}
              <td>
                <input
                  disabled={props.readOnly || (isMachineMode && row.strategy === "inherit")}
                  value={row.threshold_value}
                  onChange={(event) => updateRow(props.rows, index, "threshold_value", event.target.value, props.onChange)}
                />
              </td>
              <td>
                <select
                  disabled={props.readOnly || (isMachineMode && row.strategy === "inherit")}
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
              {isMachineMode ? (
                <td>
                  <span className={`status-badge ${thresholdEffectiveClassName(row)}`}>
                    {formatThresholdEffectiveRule(row, t)}
                  </span>
                </td>
              ) : (
                <td>
                  <input
                    checked={row.enabled}
                    disabled={props.readOnly}
                    onChange={(event) => updateRow(props.rows, index, "enabled", event.target.checked, props.onChange)}
                    type="checkbox"
                  />
                </td>
              )}
              {!isMachineMode && showSource ? <td>{formatThresholdSource(row.source, language)}</td> : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function formatThresholdEffectiveRule(row: ThresholdFormRow, t: ReturnType<typeof useI18n>["t"]) {
  const value = row.threshold_value ? `${row.threshold_value} ${row.threshold_unit}` : "-";

  if (row.strategy === "override") {
    return t("thresholdEffectiveMachineOverride", { value });
  }

  if (row.strategy === "disabled") {
    return t("thresholdEffectiveMachineDisabled");
  }

  return row.enabled
    ? t("thresholdEffectiveGlobalEnabled", { value })
    : t("thresholdEffectiveGlobalDisabled", { value });
}

function thresholdEffectiveClassName(row: ThresholdFormRow) {
  if (row.strategy === "override") {
    return "ok";
  }

  if (row.strategy === "disabled") {
    return "error";
  }

  return row.enabled ? "pending" : "idle";
}

export type { ThresholdFormRow };
