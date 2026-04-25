import type { CollectNowResponse, TrafficSample } from "../types";
import type { MachineOption } from "../lib/app-types";
import { formatTime, formatTrafficValue, machineLabel } from "../lib/app-utils";

type SamplesPageProps = {
  selectedMachineID: number | null;
  machineOptions: MachineOption[];
  samples: TrafficSample[];
  collectResults: CollectNowResponse["results"];
  onSelectMachine: (machineID: number | null) => void;
  onCollectCurrentMachine: (machineID: number) => void;
};

export default function SamplesPage(props: SamplesPageProps) {
  const filteredSamples = props.samples.filter(
    (sample) => !props.selectedMachineID || sample.machine_id === props.selectedMachineID,
  );

  return (
    <section className="panel">
      <div className="panel-header-inline">
        <h3 className="panel-title">流量样本</h3>
        <div className="header-actions">
          {props.selectedMachineID ? (
            <button
              className="secondary-button"
              onClick={() => props.onCollectCurrentMachine(props.selectedMachineID!)}
              type="button"
            >
              采集当前机器
            </button>
          ) : null}
          <select
            value={props.selectedMachineID ?? ""}
            onChange={(event) => props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
          >
            <option value="">全部机器</option>
            {props.machineOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>机器</th>
              <th>周期</th>
              <th>桶时间</th>
              <th>上行</th>
              <th>下行</th>
              <th>总量</th>
              <th>采集时间</th>
            </tr>
          </thead>
          <tbody>
            {filteredSamples.map((sample) => (
              <tr key={`${sample.id}-${sample.period_type}`}>
                <td>{machineLabel(props.machineOptions, sample.machine_id)}</td>
                <td>{sample.period_type}</td>
                <td>{formatTime(sample.bucket_time)}</td>
                <td>{formatTrafficValue(sample.upload_mb)}</td>
                <td>{formatTrafficValue(sample.download_mb)}</td>
                <td>{formatTrafficValue(sample.total_mb)}</td>
                <td>{formatTime(sample.collected_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {props.collectResults.length > 0 ? (
        <div className="list-block">
          <h4>最近手动采集结果</h4>
          {props.collectResults.map((result) => (
            <article className="card" key={`${result.machine_id}-${result.status}`}>
              <div className="card-header">
                <strong>{machineLabel(props.machineOptions, result.machine_id)}</strong>
                <span className={`status-badge ${result.status === "success" ? "ok" : "error"}`}>
                  {result.status}
                </span>
              </div>
              <p className="card-meta">样本数：{result.sample_count}</p>
              {result.error ? <p className="card-meta">错误：{result.error}</p> : null}
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}
