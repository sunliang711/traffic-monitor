type ThresholdFormRow = {
  period_type: string;
  metric_type: string;
  threshold_value: string;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
  source?: string;
};

type ThresholdEditorProps = {
  rows: ThresholdFormRow[];
  onChange: (rows: ThresholdFormRow[]) => void;
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
  return (
    <div className="table-wrapper threshold-editor">
      <table>
        <thead>
          <tr>
            <th>周期</th>
            <th>维度</th>
            <th>阈值</th>
            <th>单位</th>
            <th>启用</th>
            <th>来源</th>
          </tr>
        </thead>
        <tbody>
          {props.rows.map((row, index) => (
            <tr key={`${row.period_type}-${row.metric_type}`}>
              <td>{row.period_type}</td>
              <td>{row.metric_type}</td>
              <td>
                <input
                  value={row.threshold_value}
                  onChange={(event) => updateRow(props.rows, index, "threshold_value", event.target.value, props.onChange)}
                />
              </td>
              <td>
                <select
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
              <td>
                <input
                  checked={row.enabled}
                  onChange={(event) => updateRow(props.rows, index, "enabled", event.target.checked, props.onChange)}
                  type="checkbox"
                />
              </td>
              <td>{row.source ?? "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export type { ThresholdFormRow };
